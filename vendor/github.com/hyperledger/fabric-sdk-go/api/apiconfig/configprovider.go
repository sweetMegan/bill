/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package apiconfig

import (
	"crypto/tls"
	"crypto/x509"
	"time"
)

// Config fabric-sdk-go configuration interface
type Config interface {
	Client() (*ClientConfig, error)
	CAConfig(org string) (*CAConfig, error)
	CAServerCertPems(org string) ([]string, error)
	CAServerCertPaths(org string) ([]string, error)
	CAClientKeyPem(org string) (string, error)
	CAClientKeyPath(org string) (string, error)
	CAClientCertPem(org string) (string, error)
	CAClientCertPath(org string) (string, error)
	TimeoutOrDefault(TimeoutType) time.Duration
	MspID(org string) (string, error)
	PeerMspID(name string) (string, error)
	OrderersConfig() ([]OrdererConfig, error)
	RandomOrdererConfig() (*OrdererConfig, error)
	OrdererConfig(name string) (*OrdererConfig, error)
	PeersConfig(org string) ([]PeerConfig, error)
	PeerConfig(org string, name string) (*PeerConfig, error)
	NetworkConfig() (*NetworkConfig, error)
	NetworkPeers() ([]NetworkPeer, error)
	ChannelConfig(name string) (*ChannelConfig, error)
	ChannelPeers(name string) ([]ChannelPeer, error)
	ChannelOrderers(name string) ([]OrdererConfig, error)
	SetTLSCACertPool(*x509.CertPool)
	TLSCACertPool(certConfig ...*x509.Certificate) (*x509.CertPool, error)
	IsSecurityEnabled() bool
	SecurityAlgorithm() string
	SecurityLevel() int
	SecurityProvider() string
	Ephemeral() bool
	SoftVerify() bool
	SecurityProviderLibPath() string
	SecurityProviderPin() string
	SecurityProviderLabel() string
	KeyStorePath() string
	CAKeyStorePath() string
	CryptoConfigPath() string
	TLSClientCerts() ([]tls.Certificate, error)
}

// ConfigProvider enables creation of a Config instance
type ConfigProvider func() (Config, error)

// TimeoutType enumerates the different types of outgoing connections
type TimeoutType int

const (
	// Endorser connection timeout
	Endorser TimeoutType = iota
	// EventHub connection timeout
	EventHub
	// EventReg connection timeout
	EventReg
	// Query timeout
	Query
	// Execute timeout
	Execute
	// OrdererConnection orderer connection timeout
	OrdererConnection
	// OrdererResponse orderer response timeout
	OrdererResponse
	// DiscoveryGreylistExpiry discovery Greylist expiration period
	DiscoveryGreylistExpiry
)
