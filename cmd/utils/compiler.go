/**
 * Copyright 2024 Advanced Micro Devices, Inc.  All rights reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
**/

package utils

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/*
var tplFolder embed.FS

// Declare type pointer to a template
var temp *template.Template

// Using the init function to make sure the template is only parsed once in the program
func init() {
	// template.Must takes the reponse of template.ParseFiles and does error checking
	temp = template.Must(template.ParseFS(tplFolder, "templates/template.templ"))
}

type platformpackage struct {
	Name    string
	Kind    string
	Content bytes.Buffer
}

func shouldSkipFile(file os.DirEntry, dirPath string) bool {
	// Skip directories
	if file.IsDir() {
		return true
	}
	name := file.Name()
	// Check if file contains helm.sh/hook
	content, err := os.ReadFile(dirPath + "/" + name)
	if err != nil {
		log.Printf("Error reading file %s: %v", name, err)
		return true
	}
	if strings.Contains(string(content), "helm.sh/hook") {
		return true
	}

	return false
}

// CreateCrossplaneObject reads the output of the SplitYAML function and writes it to a file
func CreateCrossplaneObject(config Config) {
	// read a command line argument and assign it to a variable
	platformpackage := new(platformpackage)
	platformpackage.Name = config.Name
	objectFile, err := os.OpenFile("output/"+platformpackage.Name+"-object.yaml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer objectFile.Close()
	crdFile, err := os.OpenFile("output/"+platformpackage.Name+"-crd.yaml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer crdFile.Close()
	secretFile, err := os.OpenFile("output/"+platformpackage.Name+"-secret.yaml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer secretFile.Close()

	files, _ := os.ReadDir("working/" + platformpackage.Name)
	for _, file := range files {
		if shouldSkipFile(file, "working/"+platformpackage.Name) {
			continue
		}
		// split the file name to get the kind
		platformpackage.Kind = strings.Split(file.Name(), "_")[0] + "-" + strings.Split(file.Name(), "_")[1]
		// strip the .yaml extension
		platformpackage.Kind = strings.TrimSuffix(platformpackage.Kind, ".yaml")
		// Read the content of the file
		content, err := os.ReadFile("working/" + platformpackage.Name + "/" + file.Name())
		if err != nil {
			log.Fatalln(err)
		}
		lines := strings.Split(string(content), "\n")

		// Create a buffer to hold the indented content
		// var buf bytes.Buffer

		// Indent each line and write it to the buffer
		for _, line := range lines {
			platformpackage.Content.WriteString(fmt.Sprintf("%s\n", line))
		}
		// Convert the content to a string and pass it to the template
		if strings.Contains(platformpackage.Kind, "CustomResourceDefinition") {
			err = temp.Execute(crdFile, platformpackage)
		} else if strings.Contains(platformpackage.Kind, "Secret") {
			err = temp.Execute(secretFile, platformpackage)
		} else {
			err = temp.Execute(objectFile, platformpackage)
		}
		if err != nil {
			log.Fatalln(err)
		}
		// Clear the buffer
		platformpackage.Content.Reset()
	}
	removeEmptyLines("output/" + platformpackage.Name + "-object.yaml")
	removeEmptyLines("output/" + platformpackage.Name + "-crd.yaml")
	removeEmptyLines("output/" + platformpackage.Name + "-secret.yaml")
}

// CreatePackage reads the output of the SplitYAML function and writes it to a file
func CreatePackage(composition_name string, content string) {
	outfile, err := os.OpenFile("output/"+composition_name+"-packages.yml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer outfile.Close()
	lines := strings.Split(string(content), "\n")

	// Append content to outfile
	contentToAppend := strings.Join(lines, "\n")
	_, err = io.WriteString(outfile, contentToAppend)
	if err != nil {
		log.Fatalln(err)
	}
	removeEmptyLines("output/" + composition_name + "-composition.yaml")
}

func removeEmptyLines(filename string) error {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Remove empty lines
	re := regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	result := re.ReplaceAllString(string(data), "")

	// Write the result back to the file
	err = os.WriteFile(filename, []byte(result), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
