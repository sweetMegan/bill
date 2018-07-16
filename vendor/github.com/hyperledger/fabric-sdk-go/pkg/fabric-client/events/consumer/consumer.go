/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consumer

import (
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/golang/protobuf/ptypes"
	apiconfig "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config/comm"
	ccomm "github.com/hyperledger/fabric-sdk-go/pkg/config/comm"
	"github.com/hyperledger/fabric-sdk-go/pkg/config/urlutil"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	ehpb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"

	"crypto/x509"

	consumer "github.com/hyperledger/fabric-sdk-go/internal/github.com/hyperledger/fabric/events/consumer"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
)

var logger = logging.NewLogger("fabric_sdk_go")

const defaultTimeout = time.Second * 3

type eventsClient struct {
	sync.RWMutex
	peerAddress            string
	regTimeout             time.Duration
	stream                 ehpb.Events_ChatClient
	adapter                consumer.EventAdapter
	TLSCertificate         *x509.Certificate
	TLSServerHostOverride  string
	tlsCertHash            []byte
	clientConn             *grpc.ClientConn
	provider               fab.ProviderContext
	identity               fab.IdentityContext
	processEventsCompleted chan struct{}
	kap                    keepalive.ClientParameters
	failFast               bool
}

//NewEventsClient Returns a new grpc.ClientConn to the configured local PEER.
func NewEventsClient(provider fab.ProviderContext, identity fab.IdentityContext, peerAddress string, certificate *x509.Certificate,
	serverhostoverride string, regTimeout time.Duration, adapter consumer.EventAdapter,
	kap keepalive.ClientParameters, failFast bool) (fab.EventsClient, error) {
	var err error
	if regTimeout < 100*time.Millisecond {
		regTimeout = 100 * time.Millisecond
		err = errors.New("regTimeout >= 0, setting to 100 msec")
	} else if regTimeout > 60*time.Second {
		regTimeout = 60 * time.Second
		err = errors.New("regTimeout > 60, setting to 60 sec")
	}

	return &eventsClient{
		RWMutex:               sync.RWMutex{},
		peerAddress:           peerAddress,
		regTimeout:            regTimeout,
		adapter:               adapter,
		TLSCertificate:        certificate,
		TLSServerHostOverride: serverhostoverride,
		provider:              provider,
		identity:              identity,
		tlsCertHash:           ccomm.TLSCertHash(provider.Config()),
		kap:                   kap,
		failFast:              failFast,
	}, err
}

//newEventsClientConnectionWithAddress Returns a new grpc.ClientConn to the configured local PEER.
func newEventsClientConnectionWithAddress(peerAddress string, cert *x509.Certificate, serverHostOverride string,
	config apiconfig.Config, kap keepalive.ClientParameters, failFast bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	if urlutil.IsTLSEnabled(peerAddress) {
		tlsConfig, err := comm.TLSConfig(cert, serverHostOverride, config)
		if err != nil {
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	if kap.Time > 0 {
		opts = append(opts, grpc.WithKeepaliveParams(kap))
	}
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.FailFast(failFast)))

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, config.TimeoutOrDefault(apiconfig.EventHub))
	conn, err := grpc.DialContext(ctx, urlutil.ToAddress(peerAddress), opts...)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func (ec *eventsClient) send(emsg *ehpb.Event) error {
	ec.Lock()
	defer ec.Unlock()

	user := ec.identity
	payload, err := proto.Marshal(emsg)
	if err != nil {
		return errors.Wrap(err, "marshal event failed")
	}

	signingMgr := ec.provider.SigningManager()
	if signingMgr == nil {
		return errors.New("signing manager is nil")
	}

	signature, err := signingMgr.Sign(payload, user.PrivateKey())
	if err != nil {
		return errors.WithMessage(err, "sign failed")
	}
	signedEvt := &peer.SignedEvent{EventBytes: payload, Signature: signature}

	return ec.stream.Send(signedEvt)
}

// RegisterAsync - registers interest in a event and doesn't wait for a response
func (ec *eventsClient) RegisterAsync(ies []*ehpb.Interest) error {
	if ec.identity == nil {
		return errors.New("identity context is nil")
	}
	creator, err := ec.identity.Identity()
	if err != nil {
		return errors.WithMessage(err, "identity context identity retrieval failed")
	}

	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to create timestamp")
	}
	emsg := &ehpb.Event{
		Event:       &ehpb.Event_Register{Register: &ehpb.Register{Events: ies}},
		Creator:     creator,
		TlsCertHash: ec.tlsCertHash,
		Timestamp:   ts,
	}
	if err = ec.send(emsg); err != nil {
		logger.Errorf("error on Register send %s\n", err)
	}
	return err
}

// register - registers interest in a event
func (ec *eventsClient) register(ies []*ehpb.Interest) error {
	var err error
	if err = ec.RegisterAsync(ies); err != nil {
		return err
	}

	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *ehpb.Event_Register:
		case nil:
			err = errors.New("nil object for register")
		default:
			err = errors.New("invalid object for register")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(ec.regTimeout):
		err = errors.New("register timeout")
	}
	return err
}

// UnregisterAsync - Unregisters interest in a event and doesn't wait for a response
func (ec *eventsClient) UnregisterAsync(ies []*ehpb.Interest) error {
	if ec.identity == nil {
		return errors.New("identity context is required")
	}
	creator, err := ec.identity.Identity()
	if err != nil {
		return errors.WithMessage(err, "user context identity retrieval failed")
	}

	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to create timestamp")
	}
	emsg := &ehpb.Event{
		Event:       &ehpb.Event_Unregister{Unregister: &ehpb.Unregister{Events: ies}},
		Creator:     creator,
		TlsCertHash: ec.tlsCertHash,
		Timestamp:   ts,
	}

	if err = ec.send(emsg); err != nil {
		err = errors.Wrap(err, "unregister send failed")
	}

	return err
}

// unregister - unregisters interest in a event
func (ec *eventsClient) Unregister(ies []*ehpb.Interest) error {
	var err error
	if err = ec.UnregisterAsync(ies); err != nil {
		return err
	}

	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *ehpb.Event_Unregister:
		case nil:
			err = errors.New("nil object for unregister")
		default:
			err = errors.New("invalid object for unregister")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(ec.regTimeout):
		err = errors.New("unregister timeout")
	}
	return err
}

// Recv receives next event - use when client has not called Start
func (ec *eventsClient) Recv() (*ehpb.Event, error) {
	in, err := ec.stream.Recv()
	if err == io.EOF {
		// read done
		if ec.adapter != nil {
			ec.adapter.Disconnected(nil)
		}
		return nil, err
	}
	if err != nil {
		if ec.adapter != nil {
			ec.adapter.Disconnected(err)
		}
		return nil, err
	}
	return in, nil
}
func (ec *eventsClient) processEvents() error {
	defer ec.stream.CloseSend()
	defer close(ec.processEventsCompleted)

	for {
		in, err := ec.stream.Recv()
		if err == io.EOF {
			// read done.
			if ec.adapter != nil {
				ec.adapter.Disconnected(nil)
			}
			return nil
		}
		if err != nil {
			if ec.adapter != nil {
				ec.adapter.Disconnected(err)
			}
			return err
		}
		if ec.adapter != nil {
			cont, err := ec.adapter.Recv(in)
			if !cont {
				return err
			}
		}
	}
}

//Start establishes connection with Event hub and registers interested events with it
func (ec *eventsClient) Start() error {
	conn, err := newEventsClientConnectionWithAddress(ec.peerAddress, ec.TLSCertificate, ec.TLSServerHostOverride,
		ec.provider.Config(), ec.kap, ec.failFast)
	if err != nil {
		return errors.WithMessage(err, "events connection failed")
	}
	ec.clientConn = conn

	ies, err := ec.adapter.GetInterestedEvents()
	if err != nil {
		return errors.Wrap(err, "interested events retrieval failed")
	}

	if len(ies) == 0 {
		return errors.New("interested events is required")
	}

	serverClient := ehpb.NewEventsClient(conn)
	ec.stream, err = serverClient.Chat(context.Background())
	if err != nil {
		return errors.Wrap(err, "events connection failed")
	}

	if err = ec.register(ies); err != nil {
		return err
	}

	ec.processEventsCompleted = make(chan struct{})
	go ec.processEvents()

	return nil
}

//Stop terminates connection with event hub
func (ec *eventsClient) Stop() error {
	var timeoutErr error

	if ec.stream == nil {
		// in case the stream/chat server has not been established earlier, we assume that it's closed, successfully
		return nil
	}
	//this closes only sending direction of the stream; event is still there
	//read will not return an error
	err := ec.stream.CloseSend()
	if err != nil {
		return err
	}

	select {
	// Server ended its send stream in response to CloseSend()
	case <-ec.processEventsCompleted:
		// Timeout waiting for server to end stream
	case <-time.After(ec.provider.Config().TimeoutOrDefault(apiconfig.EventHub)):
		timeoutErr = errors.New("close event stream timeout")
	}

	//close  client connection
	if ec.clientConn != nil {
		err := ec.clientConn.Close()
		if err != nil {
			return err
		}
	}

	if timeoutErr != nil {
		return timeoutErr
	}

	return nil
}
