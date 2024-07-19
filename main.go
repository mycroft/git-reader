package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mycroft/git-reader/internal/git"
)

var (
	repositoryFlag string
	reference      string
	verbose        bool
	current        bool
	printReference bool
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	flag.StringVar(&repositoryFlag, "repository", cwd, "Repository path")
	flag.StringVar(&reference, "ref", "", "Object reference to dump")

	flag.BoolVar(&verbose, "verbose", false, "Verbose mode")
	flag.BoolVar(&current, "current", false, "Parse current ref")
	flag.BoolVar(&printReference, "print-ref", false, "Print ref on stderr")
}

func main() {
	var repository git.Repository
	var err error

	flag.Parse()

	if os.Getenv("REPOSITORY") != "" {
		repositoryFlag = os.Getenv("REPOSITORY")
	}

	if repository, err = git.OpenRepository(repositoryFlag); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if flag.NArg() > 0 && reference == "" {
		reference = os.Args[len(os.Args)-1]
	}

	if current {
		reference, err = repository.GetCurrentRef()
		if err != nil {
			panic(err)
		}
	}

	if printReference {
		fmt.Fprintf(os.Stderr, "ref %s\n", reference)
	}

	if reference == "" {
		for _, object := range repository.Objects {
			if verbose {
				o, err := repository.OpenObject(object.Hash)
				if err != nil {
					fmt.Println(object.Hash)
					panic(err)
				}

				fmt.Printf("%s %s %d bytes\n", o.Hash, o.Type, o.ContentLen)
			} else {
				fmt.Println(object.Hash)
			}
		}
	} else {
		o, err := repository.OpenObject(reference)
		check(err)

		builtObject := repository.ApplyDelta(o)

		if builtObject.Type == git.OBJECT_TYPE_TREE {
			tree, err := repository.ConvertTree(builtObject.Content)
			if err != nil {
				panic(err)
			}
			fmt.Print(tree)
		} else {
			fmt.Print(string(builtObject.Content))
		}
	}
}
