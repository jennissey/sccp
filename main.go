package main

import (
	"encoding/json"
	"fmt"

	// this package is called by "yaml", this is a rename operation
	// to make it clear that this is the name used in the code
	"io/ioutil"
	"net/http"
	"os"
	"path"

	yaml "gopkg.in/yaml.v2"
)

// This is supposed to take a swagger-combine config file, use the URLs
// specified in the config file to find the OAS(OpenAPI specification)
// files for the separate APIs and find the name of the API and the tags
// of the operations. Then the name of the API will be prefixed to the tags
// to make it visible, which operation belongs to which API

// My config-file lies at:
// ~/Documents/VSIS_Thesis/swagger_tools/config_original.json

// structs to store the config json data
type Config struct {
	Swagger string `json:"swagger" yaml:"swagger"`
	Info    Info   `json:"info" yaml:"info"`
	APIs    []*API `json:"apis" yaml:"apis"`
}

type Info struct {
	Title   string `json:"title" yaml:"title"`
	Version string `json:"version" yaml:"version"`
}

type API struct {
	URL   string            `json:"url" yaml:"url"`
	Tags  *Tag              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Paths map[string]string `json:"paths,omitempty" yaml:"tags,omitempty"`
}

type Tag struct {
	Rename map[string]string `json:"rename,omitempty" yaml:"tags,omitempty"`
	Add    []string          `json:"add,omitempty" yaml:"tags,omitempty"`
}

// structs to store the OpenAPI json data
type OAS struct {
	// This info contains the name of the API in Info.Title
	Info  Info            `json:"info" yaml:"info"`
	Paths map[string]Path `json:"paths" yaml:"paths"`
}

// the strings here are Methods like "get" "post" etc.
// there are more than just tags inside the paths,
// but that doesn't matter in this context
type Path map[string]Method

// Method method
// The tags inside this array are the titles that are getting
// prefixed with the API name
type Method struct {
	Tags []string `json:"tags" yaml:"tags"`
}

func main() {
	// read config file path from command line argument
	if len(os.Args) != 2 {
		fmt.Print("\n Wrong arguments\n Usage: sccp <file path to swagger-combined config file>\n exiting\n\n")
		os.Exit(1)
	}
	configFilePath := os.Args[1]
	fmt.Printf("\n Using %s\n\n", configFilePath)

	// read the config file and store it's content
	configFile, err := os.Open(configFilePath)
	if err != nil {
		fmt.Printf(" Error: %s\n Couldn't open config file\n exiting\n\n", err)
		os.Exit(1)
	}
	defer configFile.Close()
	configFileByte, err := ioutil.ReadAll(configFile)
	if err != nil {
		fmt.Printf(" Error: %s\n Couldn't read config file\n exiting\n\n", err)
		os.Exit(1)
	}

	// convert the content of the config file to a struct that can be altered
	// and then written into a new file
	var config Config
	if path.Ext(configFilePath) == ".yaml" || path.Ext(configFilePath) == ".yml" {
		yaml.Unmarshal(configFileByte, &config)
	} else {
		json.Unmarshal(configFileByte, &config)
	}

	// use the URLs to find all OAS files
	for _, api := range config.APIs {
		url := api.URL
		fmt.Printf(" Working on API %s\n", url)
		// store the content of the OAS file into a struct to make it available in go
		oasHTTP, err := http.Get(url)
		if err != nil {
			fmt.Printf(" Error: %s\n Couldn't read OpenAPI file to url %s\n exiting\n\n", err, url)
			os.Exit(1)
		}
		defer oasHTTP.Body.Close()
		oasByte, err := ioutil.ReadAll(oasHTTP.Body)
		if err != nil {
			fmt.Printf(" Error: %s\n Couldn't read OpenAPI file to url %s\n exiting\n\n", err, url)
			os.Exit(1)
		}
		var oas OAS
		if path.Ext(url) == ".yaml" {
			yaml.Unmarshal(oasByte, &oas)
		} else {
			json.Unmarshal(oasByte, &oas)
		}

		// get the name of the API to use as a prefix
		prefix := oas.Info.Title + ": "
		fmt.Printf(" Found API name: %s\n", prefix)

		// prefix tags with API name and store them in a map
		// map is automatically preventing a tag to be stored twice
		preTags := map[string]string{}
		for _, p := range oas.Paths {
			for _, method := range p {
				for _, tag := range method.Tags {
					preTags[tag] = prefix + tag
				}
			}
		}
		// if there were no tags found in the API,
		// add an Add-command in the config file of swagger-combine
		// that adds a tag into the API with the name "default"
		if len(preTags) == 0 {
			if api.Tags == nil {
				api.Tags = &Tag{}
			}
			if api.Tags.Add == nil {
				api.Tags.Add = make([]string, 0)
			}
			api.Tags.Add = append(api.Tags.Add, prefix+"default")
		}
		for _, tag := range preTags {
			fmt.Printf(" Found tag: %s\n", tag)
		}
		fmt.Println()

		// add a Rename-tag to the config file of swagger-combine for this API
		// the rename tag is a map[string]string that has to be added to the tag-struct
		// and this tag has to be added to the Tags []Tag (tag-array) of the API
		// (config.APIs[API])
		// don't delete tags that are already there
		if len(preTags) != 0 {
			if api.Tags == nil {
				api.Tags = &Tag{}
			}
			if api.Tags.Rename == nil {
				api.Tags.Rename = make(map[string]string)
			}
			for tag, preTag := range preTags {
				api.Tags.Rename[tag] = preTag
			}
		}
	}

	// write a new config file that contains all the old information and a rename
	// tag that renames every tag of the API
	newConfigByte, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf(" error: %s\nCouldn't parse config-struct to json\nexiting\n\n", err)
		os.Exit(1)
	}
	//newConfigFile, err := os.Create(configFilePath + "NEW")
	// Set the name of the new config file to old-path + New + file extension
	fileExt := path.Ext(configFilePath)
	//fileNoExt := configFilePath[0 : len(configFilePath)-len(fileExt)]
	// TODO: make it possible to choose a file name for the output file
	err = ioutil.WriteFile("combined-config"+fileExt, newConfigByte, 0644)
	if err != nil {
		fmt.Printf(" error: %s\nCouldn't write new config-file\nexiting\n\n", err)
		os.Exit(1)
	}
}
