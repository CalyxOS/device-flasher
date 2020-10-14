package flash

type DiscoveryType int

const (
	ADBDiscovered DiscoveryType = iota
	FastbootDiscovered
)
type Device struct {
	ID       string
	Codename string
	DiscoveryType DiscoveryType
}
