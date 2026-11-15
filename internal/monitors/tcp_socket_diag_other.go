//go:build !linux

package monitors

func readP2PTCPSockets(namespace string) ([]tcpSocketSample, error) {
	return nil, errTCPSocketStatsUnsupported
}
