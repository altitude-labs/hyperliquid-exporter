//go:build !linux

package monitors

func readP2PTCPSockets() ([]tcpSocketSample, error) {
	return nil, errTCPSocketStatsUnsupported
}
