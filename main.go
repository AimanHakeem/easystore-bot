package main

import (
	"fmt"
	"peak/cmd"
)

func main() {
	fmt.Print(`
 _______    ________   ________        ___    ___  ________   _________   ________   ________   _______           ________   ________   _________   
|\  ___ \  |\   __  \ |\   ____\      |\  \  /  /||\   ____\ |\___   ___\|\   __  \ |\   __  \ |\  ___ \         |\   __  \ |\   __  \ |\___   ___\ 
\ \   __/| \ \  \|\  \\ \  \___|_     \ \  \/  / /\ \  \___|_\|___ \  \_|\ \  \|\  \\ \  \|\  \\ \   __/|        \ \  \|\ /_\ \  \|\  \\|___ \  \_| 
 \ \  \_|/__\ \   __  \\ \_____  \     \ \    / /  \ \_____  \    \ \  \  \ \  \\\  \\ \   _  _\\ \  \_|/__       \ \   __  \\ \  \\\  \    \ \  \  
  \ \  \_|\ \\ \  \ \  \\|____|\  \     \/  /  /    \|____|\  \    \ \  \  \ \  \\\  \\ \  \\  \|\ \  \_|\ \       \ \  \|\  \\ \  \\\  \    \ \  \ 
   \ \_______\\ \__\ \__\ ____\_\  \  __/  / /        ____\_\  \    \ \__\  \ \_______\\ \__\\ _\ \ \_______\       \ \_______\\ \_______\    \ \__\
    \|_______| \|__|\|__||\_________\|\___/ /        |\_________\    \|__|   \|_______| \|__|\|__| \|_______|        \|_______| \|_______|     \|__|
                         \|_________|\|___|/         \|_________|                                                                                   
                                                                                                                                                    
                                                                                                                                                                                                     
`)
	cmd.ShowMenu()
}
