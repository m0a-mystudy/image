package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"image/color"
	_ "image/jpeg"

	"github.com/BurntSushi/toml"
)

type Range struct {
	Start uint32 `toml:"start"`
	End   uint32 `toml:"end"`
}
type Color struct {
	Name  string `toml:"name"`
	Red   Range  `toml:"red"`
	Blue  Range  `toml:"blue"`
	Green Range  `toml:"green"`
}

type Config struct {
	Colors []Color `toml:"colors"`
}

func loadConfig(filename string) Config {
	var conf Config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		fmt.Println(err.Error())
	}
	return conf
}

func printConfig(config Config) {
	var b bytes.Buffer
	e := toml.NewEncoder(&b)
	err := e.Encode(config)
	if err != nil {

	}
	fmt.Println(b.String())
}

type Patterns struct {
	FileName string
	Count    map[string]int
	Total    int
}

func (patterns Patterns) csvFormat(colors []Color) []string {
	ret := []string{}
	ret = append(ret, patterns.FileName)

	for _, color := range colors {
		count := patterns.Count[color.Name]
		result := float64(count) / float64(patterns.Total)
		ret = append(ret, fmt.Sprintf("%.4f%%", result*100.0))
	}
	// for _, count := range patterns.Count {
	// 	result := float64(count) / float64(patterns.Total)
	// 	ret = append(ret, fmt.Sprintf("%.4f%%", result*100.0))
	// }
	return ret
}

func (config Config) csvFormat() []string {
	ret := []string{"filename"}
	for _, v := range config.Colors {
		ret = append(ret, v.Name)
	}
	return ret
}

func detect(color color.Color, config Color) bool {
	r, g, b, _ := color.RGBA()
	r /= 256
	g /= 256
	b /= 256
	// fmt.Printf("rgb=(%d,%d,%d)\n", r, g, b)

	if r < config.Red.Start || r > config.Red.End {
		return false
	}
	if g < config.Green.Start || g > config.Green.End {
		return false
	}

	if b < config.Blue.Start || b > config.Blue.End {
		return false
	}
	return true
}
func imageDetector(fileName string, img image.Image, config Config) Patterns {
	var patterns Patterns
	patterns.Count = map[string]int{}
	patterns.FileName = fileName
	for _, color := range config.Colors {
		patterns.Count[color.Name] = 0
	}

	rect := img.Bounds()
	for i := 0; i < rect.Max.Y; i++ {
		for j := 0; j < rect.Max.X; j++ {
			patterns.Total++
			for _, v := range config.Colors {
				// fmt.Println("color = ", v.Name)
				if detect(img.At(j, i), v) {
					// fmt.Printf("%d,%d detect %s\n", j, i, v.Name)
					patterns.Count[v.Name]++
					break
				}
			}

		}
	}
	return patterns
}

func createWalker(config Config, writer *csv.Writer) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".jpg" {
			fmt.Println("detecting ....", path)
			file, err := os.Open(path)
			defer file.Close()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			img, _, _ := image.Decode(file)
			p := imageDetector(path, img, config)
			fmt.Println(p)
			writer.Write(p.csvFormat(config.Colors))
		}
		return nil
	}
}
func failOnError(err error) {
	if err != nil {
		log.Fatal("Error:", err)
	}
}

func main() {

	configName := flag.String("o", "colors.toml", "config filename")
	outputName := flag.String("c", "output.csv", "output filename")
	searchPath := flag.String("path", "./", "search file path")

	flag.Parse()
	config := loadConfig(*configName)
	file, err := os.OpenFile(*outputName, os.O_WRONLY|os.O_CREATE, 0600)
	failOnError(err)
	defer file.Close()
	err = file.Truncate(0)
	failOnError(err)

	writer := csv.NewWriter(file)
	writer.Write(config.csvFormat())
	filepath.Walk(*searchPath, createWalker(config, writer))
	writer.Flush()
}
