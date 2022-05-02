package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	libp2plog "github.com/ipfs/go-log"

	log "github.com/koinos/koinos-log-golang"
	koinosmq "github.com/koinos/koinos-mq-golang"
	"github.com/koinos/koinos-p2p/internal/node"
	"github.com/koinos/koinos-p2p/internal/options"
	"github.com/koinos/koinos-p2p/internal/rpc"
	util "github.com/koinos/koinos-util-golang"
	flag "github.com/spf13/pflag"
)

const (
	baseDirOption       = "basedir"
	amqpOption          = "amqp"
	listenOption        = "listen"
	seedOption          = "seed"
	peerOption          = "peer"
	directOption        = "direct"
	checkpointOption    = "checkpoint"
	disableGossipOption = "disable-gossip"
	forceGossipOption   = "force-gossip"
	logLevelOption      = "log-level"
	instanceIDOption    = "instance-id"
	pluginsOption       = "plugins"
)

const (
	baseDirDefault       = ".koinos"
	amqpDefault          = "amqp://guest:guest@localhost:5672/"
	listenDefault        = "/ip4/127.0.0.1/tcp/8888"
	seedDefault          = ""
	disableGossipDefault = false
	forceGossipDefault   = false
	logLevelDefault      = "info"
	instanceIDDefault    = ""
)

const (
	appName = "p2p"
	logDir  = "logs"
)

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// Set libp2p log level
	libp2plog.SetAllLoggers(libp2plog.LevelFatal)

	baseDir := flag.StringP(baseDirOption, "d", baseDirDefault, "Koinos base directory")
	amqp := flag.StringP(amqpOption, "a", "", "AMQP server URL")
	addr := flag.StringP(listenOption, "l", "", "The multiaddress on which the node will listen")
	seed := flag.StringP(seedOption, "s", "", "Seed string with which the node will generate an ID (A randomized seed will be generated if none is provided)")
	peerAddresses := flag.StringSliceP(peerOption, "p", []string{}, "Address of a peer to which to connect (may specify multiple)")
	directAddresses := flag.StringSliceP(directOption, "D", []string{}, "Address of a peer to connect using gossipsub.WithDirectPeers (may specify multiple) (should be reciprocal)")
	checkpoints := flag.StringSliceP(checkpointOption, "c", []string{}, "Block checkpoint in the form height:blockid (may specify multiple times)")
	disableGossip := flag.BoolP(disableGossipOption, "g", disableGossipDefault, "Disable gossip mode")
	forceGossip := flag.BoolP(forceGossipOption, "G", forceGossipDefault, "Force gossip mode to always be enabled")
	logLevel := flag.StringP(logLevelOption, "v", "", "The log filtering level (debug, info, warn, error)")
	instanceID := flag.StringP(instanceIDOption, "i", instanceIDDefault, "The instance ID to identify this node")
	plugins := flag.StringSliceP(pluginsOption, "P", []string{}, "Plugins allowed to use the p2p micro service")

	flag.Parse()

	*baseDir = util.InitBaseDir(*baseDir)
	util.EnsureDir(*baseDir)
	yamlConfig := util.InitYamlConfig(*baseDir)

	*amqp = util.GetStringOption(amqpOption, amqpDefault, *amqp, yamlConfig.P2P, yamlConfig.Global)
	*addr = util.GetStringOption(listenOption, listenDefault, *addr, yamlConfig.P2P, yamlConfig.Global)
	*seed = util.GetStringOption(seedOption, seedDefault, *seed, yamlConfig.P2P, yamlConfig.Global)
	*peerAddresses = util.GetStringSliceOption(peerOption, *peerAddresses, yamlConfig.P2P, yamlConfig.Global)
	*directAddresses = util.GetStringSliceOption(directOption, *directAddresses, yamlConfig.P2P, yamlConfig.Global)
	*checkpoints = util.GetStringSliceOption(checkpointOption, *checkpoints, yamlConfig.P2P, yamlConfig.Global)
	*disableGossip = util.GetBoolOption(disableGossipOption, *disableGossip, disableGossipDefault, yamlConfig.P2P, yamlConfig.Global)
	*forceGossip = util.GetBoolOption(forceGossipOption, *forceGossip, forceGossipDefault, yamlConfig.P2P, yamlConfig.Global)
	*logLevel = util.GetStringOption(logLevelOption, logLevelDefault, *logLevel, yamlConfig.P2P, yamlConfig.Global)
	*instanceID = util.GetStringOption(instanceIDOption, util.GenerateBase58ID(5), *instanceID, yamlConfig.P2P, yamlConfig.Global)
	*plugins = util.GetStringSliceOption(pluginsOption, *plugins, yamlConfig.P2P, yamlConfig.Global)

	appID := fmt.Sprintf("%s.%s", appName, *instanceID)

	// Initialize logger
	logFilename := path.Join(util.GetAppDir(*baseDir, appName), logDir, "p2p.log")
	err := log.InitLogger(*logLevel, false, logFilename, appID)
	if err != nil {
		panic(fmt.Sprintf("Invalid log-level: %s. Please choose one of: debug, info, warn, error", *logLevel))
	}

	client := koinosmq.NewClient(*amqp, koinosmq.ExponentialBackoff)
	requestHandler := koinosmq.NewRequestHandler(*amqp)

	config := options.NewConfig()

	config.NodeOptions.InitialPeers = *peerAddresses
	config.NodeOptions.DirectPeers = *directAddresses
	config.NodeOptions.Plugins = *plugins

	if *disableGossip {
		config.GossipToggleOptions.AlwaysDisable = true
	}
	if *forceGossip {
		config.GossipToggleOptions.AlwaysEnable = true
	}

	for _, checkpoint := range *checkpoints {
		parts := strings.SplitN(checkpoint, ":", 2)
		if len(parts) != 2 {
			log.Errorf("Checkpoint option must be in form blockHeight:blockID, was '%s'", checkpoint)
		}
		blockHeight, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			log.Errorf("Could not parse checkpoint block height '%s': %s", parts[0], err.Error())
		}

		// Replace with base64 later
		//blockID, err := base64.URLEncoding.DecodeString(parts[1])
		blockID, err := hex.DecodeString(parts[1])
		if err != nil {
			log.Errorf("Error decoding checkpoint block id: %s", err)
		}
		config.PeerConnectionOptions.Checkpoints = append(config.PeerConnectionOptions.Checkpoints, options.Checkpoint{BlockHeight: blockHeight, BlockID: blockID})
	}

	client.Start()

	koinosRPC := rpc.NewKoinosRPC(client)

	log.Info("Attempting to connect to block_store...")
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		val, _ := koinosRPC.IsConnectedToBlockStore(ctx)
		if val {
			log.Info("Connected")
			break
		}
	}

	log.Info("Attempting to connect to chain...")
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		val, _ := koinosRPC.IsConnectedToChain(ctx)
		if val {
			log.Info("Connected")
			break
		}
	}

	pluginsRPCs := make(map[string]*rpc.PluginRPC)

	for _, plugin := range *plugins {
		log.Info("Attempting to connect to plugin " + plugin)
		pluginRPC := rpc.NewPluginRPC(client, plugin)
		for {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			val, _ := pluginRPC.IsConnectedToPlugin(ctx)
			if val {
				log.Info("Connected")
				break
			}
		}

		pluginsRPCs[pluginRPC.Name] = pluginRPC
	}

	node, err := node.NewKoinosP2PNode(context.Background(), *addr, rpc.NewKoinosRPC(client), pluginsRPCs, requestHandler, *seed, config)
	if err != nil {
		panic(err)
	}

	requestHandler.Start()

	node.Start(context.Background())

	log.Infof("Starting node at address: %s", node.GetAddress())

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("Shutting down node...")
	// Shut the node down
	node.Close()
}
