package go209

import (
	"errors"
	"fmt"
	"os"
	"plugin"
	"strings"

	log "github.com/sirupsen/logrus"
)

// PreloadedModules is the list of .so files from the root directory we load by default
const PreloadedModules = "email-mod:slackwebhook-mod"

var modules = LoadedModules{}

// Module defines what our plugins have to define
type Module interface {
	Name() string
	EnvVars() []string
	Run(in interface{}, ev map[string]string, interactions map[string]string) error
}

// LoadedModules is a struct we use to hold modules and load modules etc
type LoadedModules struct {
	Modules []Module
}

// LoadModules loads up the plugin .so files
func (m *LoadedModules) LoadModules() error {
	modsToLoad := make(map[string]struct{})
	// var modsToLoad []string

	// Load Preloaded modules first
	for _, preLoad := range strings.Split(PreloadedModules, ":") {
		if _, ok := modsToLoad[preLoad]; !ok {
			modsToLoad[preLoad] = struct{}{}
			// modsToLoad = append(modsToLoad, preLoad)
		}
	}

	// Load dynamic modules from env var
	if len(os.Getenv("DYNAMIC_MODULES")) > 0 {
		for _, preLoad := range strings.Split(os.Getenv("DYNAMIC_MODULES"), ":") {
			if _, ok := modsToLoad[preLoad]; !ok {
				modsToLoad[preLoad] = struct{}{}
				// modsToLoad = append(modsToLoad, preLoad)
			}
		}
	}

	for preLoad := range modsToLoad {
		plug, err := plugin.Open(fmt.Sprintf("./%s.so", preLoad))
		if err != nil {
			return err
		}

		symMod, err := plug.Lookup("Module")
		if err != nil {
			return err
		}

		var mod Module
		mod, ok := symMod.(Module)
		if !ok {
			return errors.New("Unexpected type from module")
		}

		m.Modules = append(m.Modules, mod)
	}

	return nil
}

// DumpMods prints information about the loaded modules
func DumpMods() error {
	fmt.Println("Number of modules loaded: ", len(modules.Modules))
	fmt.Println("Listing loaded modules:")

	for _, mod := range modules.Modules {
		fmt.Printf("Module: %s\n", mod.Name())
		if len(mod.EnvVars()) > 0 {
			fmt.Println("EnvVars:")
			for _, ev := range mod.EnvVars() {
				adjusted := strings.ToUpper(fmt.Sprintf("%s_%s", mod.Name(), ev))
				fmt.Printf("\t%s (%s)\n", ev, adjusted)
			}
		}
	}
	return nil
}

// FetchMods returns the modules
func FetchMods() LoadedModules {
	return modules
}

func init() {
	err := modules.LoadModules()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading modules: %s", err))
	}
	// log.Info(fmt.Sprintf("*** Modules loaded successfully count: %d\n", len(modules.Modules)))
}
