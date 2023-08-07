# About The Project

This is a program that allows us to remotely control a client that connects to the application. There are also several other features, such as retrieving passwords from browsers, browsing directories, and executing commands in the command prompt.

![server_image.png](img%2Fserver_image.png)

## Why I wrote this project?
I wrote this project for fun and basic learning of the Golang language. During this time, I mainly learned about the syntax of the language itself and how to establish server -> client connections. Since I don't have much experience, I relied on popular tools like [ChatGPT](https://chat.openai.com/) and [GitHub Copilot](https://github.com/features/copilot).

## How to Install and Run the Project

Before you start, you need to have Golang installed (from the website [Golang](https://go.dev/dl/)).

1. Clone this repo `git clone https://github.com/fenix544/virus.git`

### Build Server
1. Open `server` folder and configure `config.json`
2. Then run script `build.bat` to build server module.
3. Finally, run this program using `run.bat` script.

### Build Client
1. Open `client` folder and file `client.go`.
2. Then go to 39 line and replace `HOST = "localhost"` with your address and `PORT = 2137`
3. Run `build.bat` script and wait. (If you would like to remove console after start client, you need to add `-ldflags -H=windowsgui` in `build.bat` script after `build` keyword)
4. Finally, run `client.exe` to start client.

## Features
- Downloading files,
- Downloading decrypted passwords, credit cards etc. from browsers,
- Browsing files,
- Colorful logging,
- Reconnecting client when it loses connection

## Commands
![commands.png](img%2Fcommands.png)

## License

BlackRAT is licensed under the <a href="https://mit-license.org/">MIT License</a>.