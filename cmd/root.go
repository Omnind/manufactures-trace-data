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
	"os"
	"strconv"
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

		wtime string
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
	fmt.Println("Please enter the date: ")
	fmt.Scanln(&flags.wtime)
}

func start() {
	config, err := config.NewConfigFromFile(flags.configFilePath)
	if err != nil {
		logger.Error("failed to read config file", zap.Error(err))
		return
	}

	metaList := config.StepMetas
	readFileWg := sync.WaitGroup{}
	itemPartMapChan := make(chan map[string]worker.Item, len(metaList))
	errChan := make(chan error, len(metaList))
	defer close(errChan)
	for _, meta := range metaList {
		readFileWg.Add(1)
		go file.ReadFileAndConvertToItem(meta.FilePath, config.StepOrder, meta.FullStepName, meta.ResultColumnName, meta.BeginTimeColumnName, meta.StopTimeColumnName, itemPartMapChan, errChan, &readFileWg)
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
	result := master.Run(items, config.StepOrder, 1, 2399)
	// Write result to file.
	if err := writeResult(result, config.StepOrder, flags.ouputFilePath); err != nil {
		logger.Error("Failed to write result to file", zap.Error(err))
		return
	}

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
		"LOB", "Project", "Part", "Vendor", "Date",
		"Start Station", "End Station",
		"Status",
		"Min", "10th Percentile", "25th Percentile", "50th Percentile",
		"75th Percentile", "90th Percentile", "Max",
		"Parts", "Average",
		//"1D 1 time NG", "2D 1 time NG", "3D 1 time NG", "4D 1 time NG", "5D 1 time NG", "6D 1 time NG", "7D 1 time NG", ">7D 1 time NG",
		//"1D >1 time NG", "2D >1 time NG", "3D >1 time NG", "4D >1 time NG", "5D >1 time NG", "6D >1 time NG", "7D >1 time NG", ">7D >1 time NG",
	}
	for i := 1; i <= 2400; i++ {
		s := strconv.Itoa(i)
		//m := "," + s
		header = append(header, s)
	}

	if err := writer.Write(header); err != nil {
		return err
	}
	for i := 0; i < len(fullStepOrdering); i++ {
		fromStep := fullStepOrdering[i]
		for j := i + 1; j < len(fullStepOrdering); j++ {
			toStep := fullStepOrdering[j]
			passTimeSpentIntervals := resultSet.PassStepsTimeNumber[fromStep][toStep]
			row := make([]string, 0, len(header))
			//TODO(zp): make projectCode configurable.
			row = append(row, "Watch")
			row = append(row, "N199")
			row = append(row, "HSG")
			row = append(row, "LF")
			row = append(row, flags.wtime) //flags.wtime
			row = append(row, fromStep)
			row = append(row, toStep) //toStep
			row = append(row, "passed")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			for _, interval := range passTimeSpentIntervals {
				row = append(row, strconv.Itoa(interval))
			}
			if err := writer.Write(row); err != nil {
				return err
			}
			failTimeSpentIntervals := resultSet.FailStepsTimeNumber[fromStep][toStep]
			row = make([]string, 0, len(header))
			//TODO(zp): make projectCode configurable.
			row = append(row, "Watch")
			row = append(row, "N199")
			row = append(row, "HSG")
			row = append(row, "LF")
			row = append(row, flags.wtime) //flags.wtime
			row = append(row, fromStep)
			row = append(row, toStep) //toStep
			row = append(row, "failed")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			row = append(row, "0")
			for _, interval := range failTimeSpentIntervals {
				row = append(row, strconv.Itoa(interval))
			}
			//for i := 1; i < 17; i++ {
			//	row = append(row, "0")
			//}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}
	return nil
}
