package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"szesto.com/mqrunner/mqsc"
)

func FetchMergeConfigFiles(fetchconf FetchConfig, mqscicpath, qminipath string) error {

	fetchdir, err := ioutil.TempDir("", "git-fetch")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(fetchdir) }()

	err = GitFetch(fetchdir, fetchconf)
	if err != nil {
		return err
	}

	log.Printf("fetched git repo %s to %s\n", fetchconf.Url, fetchdir)

	dir := fetchdir
	if len(fetchconf.Dir) > 0 {
		dir = filepath.Join(fetchdir, fetchconf.Dir)

		if _, err := os.Stat(dir); err != nil {
			return err
		}
	}

	// merge ini files
	err = QminiMerge(dir, qminipath)
	if err != nil {
		return err
	}

	// merge mqsc files
	err = MqscMerge(dir, mqscicpath)
	if err != nil {
		return err
	}

	// merge mq yaml files
	err = MqYamlMerge(dir, mqscicpath)
	if err != nil {
		return err
	}

	return nil
}

func QminiMerge(dir, outfile string) error {

	// find *.ini files
	mqinifiles, err := ReadDir(dir, "ini")
	if err != nil {
		return err
	}

	if len(mqinifiles) == 0 {
		return nil
	}

	// for each mqsc file, merge into output file
	for _, mqinifile := range mqinifiles {
		err = AppendFile(mqinifile, outfile, "#*")
		if err != nil {
			// print error message and continue
			log.Printf("mqini-merge error, file %s : %v\n", mqinifile, err)
		}
	}

	return nil
}

func MqscMerge(dir string, outfile string) error {

	// find *.mqsc files in input directory
	mqscfiles, err := ReadDir(dir, "mqsc")
	if err != nil {
		return err
	}

	if len(mqscfiles) == 0 {
		return nil
	}

	// for each mqsc file, merge into output file
	for _, mqscfile := range mqscfiles {
		err = AppendFile(mqscfile, outfile, "*")
		if err != nil {
			// print error message and continue
			log.Printf("mqsc-merge error, file %s : %v\n", mqscfile, err)
		}
	}

	return nil
}

func MqYamlMerge(dir string, outfile string) error {

	// find *.yaml files
	yamlfiles, err := ReadDir(dir, "yaml")
	if err != nil {
		return err
	}

	if len(yamlfiles) == 0 {
		return nil
	}

	// create temp dir for mqyam output files
	yamloutdir, err := ioutil.TempDir("", "mqyaml-out")
	if err != nil {
		return err
	}

	// keep yaml output files
	//	defer func() { _ = os.RemoveAll(yamloutdir) }()

	// for each mq yaml file:
	for _, yamlfile := range yamlfiles {
		yamlout := fmt.Sprintf("%s.mqsc", filepath.Join(yamloutdir, path.Base(yamlfile)))

		// output mqsc file
		err = mqsc.Outputmqsc(yamlfile, yamlout)
		if err != nil {
			log.Printf("mq-yaml-merge, error converting mq yaml file to mqsc, file %s, %v\n", yamlfile, err)
			continue
		}

		// merge mqsc file into the output file
		err = AppendFile(yamlout, outfile, "*")
		if err != nil {
			// print error message and continue
			log.Printf("mq-yaml-merge error, file %s : %v\n", yamlout, err)
		}
	}

	return nil
}

func AppendFile(infile, outfile, separator string) error {

	databytes, err := os.ReadFile(infile)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() } ()

	// write mqsc separator
	_, err = f.WriteString(fmt.Sprintf("%s file '%s'\n", separator, infile))
	if err != nil {
		return err
	}

	// write input file
	_, err = f.Write(databytes)
	if err != nil {
		return err
	}

	return nil
}

func ReadDir(dir, suffix string) ([]string, error) {
	var files []string

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil

		} else {
			return nil, err
		}
	}

	for _, entry := range entries {
		if ! entry.IsDir() {
			if strings.HasSuffix(entry.Name(), suffix) {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	}

	return files, nil
}
