package options

// Config is the entire configuration file
type Config struct {
	NodeOptions            NodeOptions
	SyncManagerOptions     SyncManagerOptions
	BdmiProviderOptions    BdmiProviderOptions
	DownloadManagerOptions DownloadManagerOptions
	PeerHandlerOptions     PeerHandlerOptions
	SyncServiceOptions     SyncServiceOptions
	BlacklistOptions       BlacklistOptions
}

func NewConfig() *Config {
	config := Config{
		NodeOptions:            *NewNodeOptions(),
		SyncManagerOptions:     *NewSyncManagerOptions(),
		BdmiProviderOptions:    *NewBdmiProviderOptions(),
		DownloadManagerOptions: *NewDownloadManagerOptions(),
		PeerHandlerOptions:     *NewPeerHandlerOptions(),
		SyncServiceOptions:     *NewSyncServiceOptions(),
		BlacklistOptions:       *NewBlacklistOptions(),
	}
	return &config
}

func (config *Config) SetEnableDebugMessages(enableDebugMessages bool) {
	config.NodeOptions.EnableDebugMessages = enableDebugMessages
	config.SyncManagerOptions.EnableDebugMessages = enableDebugMessages
	config.BdmiProviderOptions.EnableDebugMessages = enableDebugMessages
	config.DownloadManagerOptions.EnableDebugMessages = enableDebugMessages
	config.PeerHandlerOptions.EnableDebugMessages = enableDebugMessages
	config.SyncServiceOptions.EnableDebugMessages = enableDebugMessages
}
