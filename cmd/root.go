/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"appledata/cmd/file"
	"appledata/config"
	"appledata/logger"
	"appledata/worker"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	greetingBanner = `
 █████╗ ██████╗ ██████╗ ██╗     ███████╗    ██████╗  █████╗ ████████╗ █████╗ 
██╔══██╗██╔══██╗██╔══██╗██║     ██╔════╝    ██╔══██╗██╔══██╗╚══██╔══╝██╔══██╗
███████║██████╔╝██████╔╝██║     █████╗█████╗██║  ██║███████║   ██║   ███████║
██╔══██║██╔═══╝ ██╔═══╝ ██║     ██╔══╝╚════╝██║  ██║██╔══██║   ██║   ██╔══██║
██║  ██║██║     ██║     ███████╗███████╗    ██████╔╝██║  ██║   ██║   ██║  ██║
╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝╚══════╝    ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
																			 
	`
)

var (
	flags struct {
		// configFilePath is the path to the config file that we will read.
		configFilePath string
		// inputFilesPath is the folder to the input file that we will read.
		inputFilesPath string
		// output file path is the path to the output file that we will write.
		ouputFilePath string
		// debug mode is a flag that will enable debug mode.
		debug bool
	}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apd [command]",
	Short: "Apple Data CLI",
	Long:  `Apple Data CLI is a command line tool to process Apple part data.`,
	Run: func(cmd *cobra.Command, args []string) {
		printGreeting()
		// We should modify the logger level to debug if the debug flag is set.
		if flags.debug {
			logger.SetLevel(zap.DebugLevel)
		}
		start()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.appledata.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVar(&flags.configFilePath, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&flags.inputFilesPath, "input", "", "input files folder")
	rootCmd.PersistentFlags().StringVar(&flags.ouputFilePath, "out", "", "output file path")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "debug mode")
}

func printGreeting() {
	fmt.Println(greetingBanner)
}

func start() {
	config, err := config.NewConfigFromFile(flags.configFilePath)
	if err != nil {
		logger.Error("failed to read config file", zap.Error(err))
		return
	}
	// We should read all the csv file in the input folder.
	fileInfos, err := ioutil.ReadDir(flags.inputFilesPath)
	if err != nil {
		logger.Error("Failed to read input folder", zap.String("folder path", flags.inputFilesPath), zap.Error(err))
		return
	}

	readFileWg := sync.WaitGroup{}
	stepShortName2FilePath := make(map[string]string)
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".csv") {
			// It's ok to skip the folder because we only need to read the csv file.
			continue
		}
		for _, stepShortName := range config.StepOrderShort {
			need := convertStepNameToFileName(stepShortName)
			if fileInfo.Name() == need {
				stepShortName2FilePath[stepShortName] = filepath.Join(flags.inputFilesPath, fileInfo.Name())
				break
			}
		}
	}
	itemPartMapChan := make(chan map[string]worker.Item, len(stepShortName2FilePath))
	errChan := make(chan error, len(stepShortName2FilePath))
	defer close(errChan)
	for stepShortName, filePath := range stepShortName2FilePath {
		readFileWg.Add(1)
		go file.ReadFileAndConvertToItem(filePath, config.StepOrder, stepShortName, config.NameMap[stepShortName], itemPartMapChan, errChan, &readFileWg)
		time.Sleep(300 * time.Millisecond)
	}
	readFileWg.Wait()
	close(itemPartMapChan)

	select {
	// TODO(zp): handle a lot of errors.
	case err := <-errChan:
		logger.Error("Failed to read file", zap.Error(err))
		return
	default:
	}

	// Tidy item map.
	itemMap := make(map[string]worker.Item)
	for {
		if itemPartMap, ok := <-itemPartMapChan; ok {
			for _, item := range itemPartMap {
				// each item includes one step
				itemSerialNumber := item.GetSerialNumber()
				if _, ok := itemMap[itemSerialNumber]; !ok {
					itemMap[itemSerialNumber] = item
					continue
				}
				allSteps := item.GetAllSteps()
				for _, step := range allSteps {
					// TODO(zp): tidy here.
					itemMap[itemSerialNumber].(*worker.PartItem).SetStep(step.GetName(), step)
				}
			}
		} else {
			break
		}
	}
	// Run map-reduce
	master := worker.NewDefaultMaster()
	items := flattenItemMap(itemMap)
	result := master.Run(items, config.StepOrder, 24, 7)
	// Write result to file.
	if err := writeResult(result, config.StepOrder, flags.ouputFilePath); err != nil {
		logger.Error("Failed to write result to file", zap.Error(err))
		return
	}

}

func convertStepNameToFileName(stepShortName string) string {
	specs := []string{
		"raw-data",
		"download",
		"N199",
		"Housing TI-LDG",
		stepShortName,
	}
	return strings.Join(specs, "-") + ".csv"
}

func flattenItemMap(itemMap map[string]worker.Item) []worker.Item {
	items := make([]worker.Item, 0, len(itemMap))
	for _, item := range itemMap {
		items = append(items, item)
	}
	return items
}

func writeResult(resultSet *worker.ResultSet, fullStepOrdering []string, outputFilePath string) error {
	file, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ','
	writer.UseCRLF = true
	// We need to write the output csv file like format:
	// ProjectCode,FromStep,ToStep,1,2,3,4,5,6,7,> 7
	// We write the header first.
	header := []string{
		"ProjectCode", "FromStep", "ToStep",
		"1-Days", "2-Days", "3-Days",
		"4-Days", "5-Days", "6-Days",
		"7-Days", ">7-Days",
	}

	if err := writer.Write(header); err != nil {
		return err
	}
	for i := 0; i < len(fullStepOrdering); i++ {
		fromStep := fullStepOrdering[i]
		for j := i; j < len(fullStepOrdering); j++ {
			toStep := fullStepOrdering[j]
			timeSpentIntervals := resultSet.StepsTimeNumber[fromStep][toStep]
			row := make([]string, 0, len(header))
			//TODO(zp): make projectCode configurable.
			row = append(row, "N199")
			row = append(row, fromStep)
			row = append(row, toStep)
			for _, interval := range timeSpentIntervals {
				row = append(row, strconv.Itoa(interval))
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}
	return nil
}
