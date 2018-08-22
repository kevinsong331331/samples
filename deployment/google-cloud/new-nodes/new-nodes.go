package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/dispatchlabs/disgo/commons/types"

	logger "github.com/nic0lae/golog"
	gologC "github.com/nic0lae/golog/contracts"
	gologM "github.com/nic0lae/golog/modifiers"
	gologP "github.com/nic0lae/golog/persisters"
)

// SeedsCount - nr of SEED(s) to spawn
const SeedsCount = 0

// DelegatesCount - nr of DELEGATE(s) to spawn
const DelegatesCount = 0

// NodesCount - nr of NODE(s) to spawn
const NodesCount = 0

// NamePrefix - VM name prefix
const NamePrefix = "test-net-2-1"

// VMConfig - config for a VM in Google Cloud
type VMConfig struct {
	ImageProject        string
	ImageFamily         string
	MachineType         string
	Tags                string
	NamePrefix          string
	ScriptPrefixURL     string
	ScriptNewNode       string
	ScriptUpdateNode    string
	ScriptNewScandis    string
	ScriptUpdateScandis string
	CodeBranch          string
}

func main() {
	// Config Logger
	var inmemoryLogger = gologM.NewInmemoryLogger(
		gologM.NewSimpleFormatterLogger(
			gologM.NewMultiLogger(
				[]gologC.Logger{
					gologP.NewConsoleLogger(),
					gologP.NewFileLogger("new-nodes.log"),
				},
			),
		),
	)
	logger.StoreSingleton(
		logger.NewLogger(
			inmemoryLogger,
		),
	)

	fmt.Println("Running, please wait...")

	// Set defaults
	var defaultVMConfig = VMConfig{
		ImageProject:    "debian-cloud",
		ImageFamily:     "debian-9",
		MachineType:     "n1-standard-2",
		Tags:            "disgo-node",
		NamePrefix:      NamePrefix + "-seed",
		ScriptPrefixURL: "https://raw.githubusercontent.com/dispatchlabs/samples/dev/deployment",
		ScriptNewNode:   "vm-debian9-new-node.sh",
		CodeBranch:      "master",
	}

	var defaultNodeConfig = types.Config{
		HttpEndpoint: &types.Endpoint{
			Host: "0.0.0.0",
			Port: 1975,
		},
		GrpcEndpoint: &types.Endpoint{
			Host: "0.0.0.0",
			Port: 1973,
		},
		GrpcTimeout:        5,
		UseQuantumEntropy:  false,
		SeedEndpoints:      []*types.Endpoint{},
		GenesisTransaction: `{"hash":"a48ff2bd1fb99d9170e2bae2f4ed94ed79dbc8c1002986f8054a369655e29276","type":0,"from":"e6098cc0d5c20c6c31c4d69f0201a02975264e94","to":"3ed25f42484d517cdfc72cafb7ebc9e8baa52c2c","value":10000000,"data":"","time":0,"signature":"03c1fdb91cd10aa441e0025dd21def5ebe045762c1eeea0f6a3f7e63b27deb9c40e08b656a744f6c69c55f7cb41751eebd49c1eedfbd10b861834f0352c510b200","hertz":0,"fromName":"","toName":""}`,
	}

	// Create SEED VMs
	seedVMConfig := defaultVMConfig
	seedVMConfig.NamePrefix = NamePrefix + "-seed"

	createNodeVMs(SeedsCount, &seedVMConfig, &defaultNodeConfig)

	var seedEndpoints = getEndpoints(SeedsCount, NamePrefix+"-seed")

	// Create DELEGATE VMs
	delegateVMConfig := defaultVMConfig
	delegateVMConfig.NamePrefix = NamePrefix + "-delegate"

	delegateConfig := defaultNodeConfig
	delegateConfig.SeedEndpoints = seedEndpoints

	createNodeVMs(DelegatesCount, &delegateVMConfig, &delegateConfig)

	// Create NODE VMs
	nodeVMConfig := defaultVMConfig
	nodeVMConfig.NamePrefix = NamePrefix + "-node"

	delegateNodeConfig := defaultNodeConfig
	delegateNodeConfig.SeedEndpoints = seedEndpoints

	createNodeVMs(NodesCount, &nodeVMConfig, &delegateNodeConfig)

	// Create Scandis VM
	var scandisVMConfig = &VMConfig{
		ImageProject:     "debian-cloud",
		ImageFamily:      "debian-9",
		MachineType:      "n1-standard-2",
		Tags:             "http-server,https-server",
		NamePrefix:       NamePrefix + "-scandis",
		ScriptPrefixURL:  "https://raw.githubusercontent.com/dispatchlabs/samples/dev/deployment",
		ScriptNewScandis: "vm-debian9-new-scandis.sh",
		CodeBranch:       "dev",
	}

	createScandisVMs(scandisVMConfig, seedVMConfig.NamePrefix+"-0")

	fmt.Println("Done.")

	// Dump log
	(inmemoryLogger.(*gologM.InmemoryLogger)).Flush()
}

// Get the underlying OS command shell
func getOSC() string {

	osc := "sh"
	if runtime.GOOS == "windows" {
		osc = "cmd"
	}

	return osc
}

// Get the shell/command startup option to execute commands
func getOSE() string {

	ose := "-c"
	if runtime.GOOS == "windows" {
		ose = "/c"
	}
	return ose
}

func createNodeVMs(count int, vmConfig *VMConfig, nodeConfig *types.Config) {
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		var vmName = fmt.Sprintf("%s-%d", vmConfig.NamePrefix, i)

		// Command to CREATE new VM Instance
		var createVM = fmt.Sprintf(
			"gcloud compute instances create %s --image-project %s --image-family %s --machine-type %s --tags %s",
			vmName,
			vmConfig.ImageProject,
			vmConfig.ImageFamily,
			vmConfig.MachineType,
			vmConfig.Tags,
		)

		// Command to DOWNLOAD BASH scripts to the newly created VM
		var downloadScriptFiles = fmt.Sprintf(
			"gcloud compute ssh %s --command 'curl %s/%s -o %s'",
			vmName,
			vmConfig.ScriptPrefixURL,
			vmConfig.ScriptNewNode,
			vmConfig.ScriptNewNode,
		)

		// Commands to RUN scripts
		var execScript = fmt.Sprintf(
			"gcloud compute ssh %s --command 'bash %s %s'",
			vmName,
			vmConfig.ScriptNewNode,
			vmConfig.CodeBranch,
		)

		// RUN VM creation in PARALLEL
		// Run COMMANDS inside the VM in SEQUENTIAL order
		wg.Add(1)
		go func(vritualMachineName string, disgoConfig *types.Config, cmds ...string) {
			logger.Instance().LogInfo(vritualMachineName, 0, "~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~")
			logger.Instance().LogInfo(vritualMachineName, 0, "DEPLOY START")

			for _, cmd := range cmds {
				logger.Instance().LogInfo(vritualMachineName, 4, cmd)
				exec.Command(getOSC(), getOSE(), cmd).Run()
			}

			var thisVMIP = getVMIP(vritualMachineName)
			disgoConfig.HttpEndpoint.Host = thisVMIP
			disgoConfig.GrpcEndpoint.Host = thisVMIP

			replaceConfigFileOnVM(vritualMachineName, disgoConfig)

			wg.Done()
		}(vmName, nodeConfig, createVM, downloadScriptFiles, execScript)
	}

	wg.Wait()
}

func createScandisVMs(vmConfig *VMConfig, seedVmName string) {
	var wg sync.WaitGroup

	// Command to CREATE new VM Instance
	var createVM = fmt.Sprintf(
		"gcloud compute instances create %s --image-project %s --image-family %s --machine-type %s --tags %s",
		vmConfig.NamePrefix,
		vmConfig.ImageProject,
		vmConfig.ImageFamily,
		vmConfig.MachineType,
		vmConfig.Tags,
	)

	// Command to DOWNLOAD BASH scripts to the newly created VM
	var downloadScriptFiles = fmt.Sprintf(
		"gcloud compute ssh %s --command 'curl %s/%s -o %s'",
		vmConfig.NamePrefix,
		vmConfig.ScriptPrefixURL,
		vmConfig.ScriptNewScandis,
		vmConfig.ScriptNewScandis,
	)

	// Commands to RUN scripts
	var seedVMIP = getVMIP(seedVmName)

	var execScript = fmt.Sprintf(
		"gcloud compute ssh %s --command 'bash %s %s %s:1975'",
		vmConfig.NamePrefix,
		vmConfig.ScriptNewScandis,
		vmConfig.CodeBranch,
		seedVMIP,
	)

	// RUN VM creation in PARALLEL
	// Run COMMANDS inside the VM in SEQUENTIAL order
	wg.Add(1)
	go func(vritualMachineName string, cmds ...string) {
		logger.Instance().LogInfo(vritualMachineName, 0, "~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~ ~~~~")
		logger.Instance().LogInfo(vritualMachineName, 0, "DEPLOY SCANDIS START")

		for _, cmd := range cmds {
			logger.Instance().LogInfo(vritualMachineName, 4, cmd)
			exec.Command(getOSC(), getOSE(), cmd).Run()
		}

		wg.Done()
	}(vmConfig.NamePrefix, createVM, downloadScriptFiles, execScript)

	wg.Wait()
}

func getVMIP(vmName string) string {
	out, err := exec.Command(getOSC(), getOSE(), fmt.Sprintf(
		"gcloud compute instances describe %s | grep natIP:",
		vmName,
	)).Output()

	if err == nil {
		var outputAsString = string(out)
		outputAsString = strings.Replace(outputAsString, "natIP: ", "", -1)
		return strings.TrimSpace(outputAsString)
	}

	return ""
}

func getEndpoints(count int, namePrefix string) []*types.Endpoint {
	var endpoints = []*types.Endpoint{}

	for i := 0; i < count; i++ {
		var vmName = fmt.Sprintf("%s-%d", namePrefix, i)

		var ip = getVMIP(vmName)
		if ip != "" {
			endpoints = append(endpoints, &types.Endpoint{Host: ip, Port: 1973})
		}
	}

	return endpoints
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func replaceConfigFileOnVM(vmName string, disgoConfig *types.Config) {
	var configFileName = randString(20) + ".json"
	file, error := os.Create(configFileName)
	if error == nil {
		bytes, error := json.Marshal(&disgoConfig)
		if error == nil {
			// Save JSON config to a temp file
			fmt.Fprintf(file, string(bytes))
			file.Close()

			var fullFileName, _ = filepath.Abs(configFileName)

			// Upload temp file to the VM
			var cmd1 = fmt.Sprintf("gcloud compute scp %s %s:~/config.json", fullFileName, vmName)
			var cmd2 = fmt.Sprintf("gcloud compute ssh %s --command 'sudo mv ~/config.json /go-binaries/config/ && sudo chown -R dispatch-services:dispatch-services /go-binaries'", vmName)
			var cmd3 = fmt.Sprintf("gcloud compute ssh %s --command 'sudo sudo systemctl restart dispatch-disgo-node'", vmName)

			logger.Instance().LogInfo(vmName, 4, cmd1)
			logger.Instance().LogInfo(vmName, 4, cmd2)
			logger.Instance().LogInfo(vmName, 4, cmd3)

			exec.Command(getOSC(), getOSE(), cmd1).Run()
			exec.Command(getOSC(), getOSE(), cmd2).Run()
			exec.Command(getOSC(), getOSE(), cmd3).Run()

			// Remove temp file
			os.Remove(configFileName)
		}
	}
}
