package cmd

import (
	"fmt"
	"peak/tasks"

	"github.com/manifoldco/promptui"
)

func ShowMenu() {
	prompt := promptui.Select{
		Label: "Select an option",
		Items: []string{"Run Tasks", "Test Proxies", "Exit"},
	}

	for {
		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		switch result {
		case "Run Tasks":
			tasks.RunTasks()
		case "Test Proxies":
			tasks.TestProxies()
		case "Exit":
			fmt.Println("Exiting...")
			return
		}
	}
}
