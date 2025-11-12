package main

import (
	"log"
	"maps"
	"slices"
	"sort"
)

func sortUniqueKeys(maps ...map[string]string) (unique []string) {
	combined := make(map[string]struct{})
	for _, m := range maps {
		for key := range m {
			combined[key] = struct{}{}
		}
	}
	for key := range combined {
		unique = append(unique, key)
	}
	sort.Strings(unique)
	return unique
}

func printMapKeys(m map[string]string, preamble string) {
	printStrings(slices.Collect(maps.Keys(m)), preamble)
}

func printStrings(paths []string, preamble string) {
	log.Printf(preamble, len(paths))
	if len(paths) == 0 {
		return
	}
	for _, path := range paths {
		log.Println("  " + path)
	}
}
