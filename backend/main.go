package main

import (
	"errors"
	"fmt"
	"github.com/JackC/cli"
	"github.com/JackC/pgx"
	"github.com/vaughan0/go-ini"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const version = "0.6.0pre"

var config struct {
	configPath    string
	listenAddress string
	listenPort    string
	staticURL     string
}

func extractConnConfig(file ini.File) (connConfig pgx.ConnConfig, err error) {
	connConfig.Host, _ = file.Get("database", "host")
	connConfig.Socket, _ = file.Get("database", "socket")
	if connConfig.Host == "" && connConfig.Socket == "" {
		err = errors.New("Config must contain database.host or database.socket but it does not")
		return
	}

	if p, ok := file.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		connConfig.Port = uint16(n)
		if err != nil {
			return connConfig, err
		}
	}

	var ok bool

	if connConfig.Database, ok = file.Get("database", "database"); !ok {
		err = errors.New("Config must contain database.database but it does not")
		return
	}
	if connConfig.User, ok = file.Get("database", "user"); !ok {
		err = errors.New("Config must contain database.user but it does not")
		return
	}
	connConfig.Password, _ = file.Get("database", "password")
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "tpr"
	app.Usage = "The Pithy Reader RSS Aggregator"
	app.Version = version
	app.Author = "Jack Christensen"
	app.Email = "jack@jackchristensen.com"

	app.Commands = []cli.Command{
		{
			Name:        "server",
			ShortName:   "s",
			Usage:       "run the server",
			Synopsis:    "[command options]",
			Description: "run the tpr server",
			Flags: []cli.Flag{
				cli.StringFlag{"address, a", "127.0.0.1", "address to listen on"},
				cli.StringFlag{"port, p", "8080", "port to listen on"},
				cli.StringFlag{"config, c", "tpr.conf", "path to config file"},
				cli.StringFlag{"static-url", "", "reverse proxy static asset requests to URL"},
			},
			Action: Serve,
		},
		{
			Name:        "reset-password",
			Usage:       "reset a user's password",
			Synopsis:    "[command options] username",
			Description: "reset a user's password",
			Flags: []cli.Flag{
				cli.StringFlag{"config, c", "tpr.conf", "path to config file"},
				cli.StringFlag{"password, p", "", "password to set"},
			},
			Action: ResetPassword,
		},
	}

	app.Run(os.Args)

}

func configure(c *cli.Context) (repository, error) {
	var err error

	config.listenAddress = c.String("address")
	config.listenPort = c.String("port")
	config.configPath = c.String("config")
	config.staticURL = c.String("static-url")

	if config.configPath, err = filepath.Abs(config.configPath); err != nil {
		return nil, fmt.Errorf("Invalid config path: %v", err)
	}

	file, err := ini.LoadFile(config.configPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config file: %v", err)
	}

	var ok bool

	if !c.IsSet("address") {
		if config.listenAddress, ok = file.Get("server", "address"); !ok {
			return nil, errors.New("Missing server address")
		}
	}

	if !c.IsSet("port") {
		if config.listenPort, ok = file.Get("server", "port"); !ok {
			return nil, errors.New("Missing server port")
		}
	}

	poolConfig := pgx.ConnPoolConfig{MaxConnections: 10, AfterConnect: afterConnect}
	if poolConfig.ConnConfig, err = extractConnConfig(file); err != nil {
		return nil, fmt.Errorf("Error reading database connection: %v", err.Error())
	}
	poolConfig.Logger = &PackageLogger{logger: logger, pkg: "pgx"}

	repo, err := NewPgxRepository(poolConfig)
	if err != nil {
		return nil, fmt.Errorf("Unable to create pgx repository: %v", err)
	}

	return repo, nil
}

func Serve(c *cli.Context) {
	repo, err := configure(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiHandler := NewAPIHandler(repo)
	http.Handle("/api/", http.StripPrefix("/api", apiHandler))

	if config.staticURL != "" {
		staticURL, err := url.Parse(config.staticURL)
		if err != nil {
			logger.Fatal("tpr", fmt.Sprintf("Bad static-url: %v", err))
			os.Exit(1)
		}
		http.Handle("/", httputil.NewSingleHostReverseProxy(staticURL))
	}

	listenAt := fmt.Sprintf("%s:%s", config.listenAddress, config.listenPort)
	fmt.Printf("Starting to listen on: %s\n", listenAt)

	feedUpdater := NewFeedUpdater(repo)
	go feedUpdater.KeepFeedsFresh()

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
}

func ResetPassword(c *cli.Context) {
	if len(c.Args()) != 1 {
		cli.ShowCommandHelp(c, c.Command.Name)
		os.Exit(1)
	}

	name := c.Args()[0]

	repo, err := configure(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	user, err := repo.GetUserByName(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	password, err := genRandPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	update := &User{}
	update.SetPassword(password)

	err = repo.UpdateUser(user.ID.MustGet(), update)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("User:", name)
	fmt.Println("Password:", password)
}
