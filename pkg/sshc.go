package pinger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var terminalModes = ssh.TerminalModes{
	ssh.ECHO:          1,     // enable echoing (different from the example in docs)
	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
}

type SshCluster struct {
	Name  string   `json:"name"`
	Nodes []string `json:"nodes"`
}

type SshEndpointData struct {
	Cluster  string `json:"cluster"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Target   string `json:"target"`
	Username string `json:"username"`
	Password string `json:"password"`
	Key      string `json:"key"`
}

type sshClient struct {
	addr    string
	user    string
	secret  string
	keyfile string
	keyRaw  string

	client  *ssh.Client
	sess    *ssh.Session
	sessIn  io.WriteCloser
	sessOut io.Reader
}

var mcpClient *client.Client

func GetAllSshNodes() ([]SshCluster, error) {
	log.Info("sshClient: GetAllNodes")
	var err error

	if mcpClient == nil {
		return nil, fmt.Errorf("Plugins were not configured.")
	}

	clusters, err := callGetAllNodesPlugin()
	if err != nil {
		log.Error("sshClient: callGetAllNodesPlugin:", err)
		return nil, err
	}

	return clusters, nil
}

func NewSshClient(sshTarget string) (*sshClient, error) {
	log.Info("sshClient: NewSshClient:", sshTarget)

	var err error
	if mcpClient == nil {
		return nil, fmt.Errorf("Plugins were not configured.")
	}

	sshEndpoint, err := callSshEndpointPlugin(sshTarget)
	if err != nil {
		log.Error("sshClient: callSshEndpointPlugin:", err)
		return nil, err
	}

	c := &sshClient{
		addr:    sshEndpoint.Target, //fmt.Sprintf("%s:%d", sshEndpoint.Host, sshEndpoint.Port),
		user:    sshEndpoint.Username,
		secret:  sshEndpoint.Password,
		keyfile: "",
		keyRaw:  sshEndpoint.Key,
	}

	// Log the SSH connection details
	log.WithFields(log.Fields{
		"sshTarget": sshTarget,
		"host":      sshEndpoint.Host,
		"port":      sshEndpoint.Port,
		"username":  sshEndpoint.Username,
	}).Info("SSH connection details")

	err = c.connect()
	return c, err
}

func (c *sshClient) connect() error {
	var err error
	wdSize := &windowSize{
		Rows: 24,
		Cols: 80,
		X:    0,
		Y:    0,
	}

	var auth ssh.AuthMethod
	if c.secret != "" {
		auth = ssh.Password(c.secret)
	} else if c.keyfile != "" {
		key, err := os.ReadFile(c.keyfile)
		if err != nil {
			log.Error("sshClient: os.ReadFile:", err)
			return err
		}
		privateKey, err := ssh.ParsePrivateKey(key)
		if err != nil {
			log.Error("sshClient: ssh.ParsePrivateKey:", err)
			return err
		}
		auth = ssh.PublicKeys(privateKey)
	} else {
		privateKey, err := ssh.ParsePrivateKey([]byte(c.keyRaw))
		if err != nil {
			log.Error("sshClient: ssh.ParsePrivateKey:", err)
			return err
		}
		auth = ssh.PublicKeys(privateKey)

	}

	config := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{auth},
		// InsecureIgnoreHostKey returns a function
		// that can be used for ClientConfig.HostKeyCallback
		// to accept any host key.
		// It should not be used for production code.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	c.client, err = ssh.Dial("tcp", c.addr, config)
	if err != nil {
		log.Info("sshClient: ssh.Dial:", err)
		return err
	}

	c.sess, err = c.client.NewSession()
	if err != nil {
		log.Error("sshClient: client.NewSession:", err)
		return err
	}

	c.sess.Stderr = os.Stderr
	c.sessOut, err = c.sess.StdoutPipe()
	if err != nil {
		log.Error("sshClient: session.StdoutPipe:", err)
		return err
	}

	c.sessIn, err = c.sess.StdinPipe()
	if err != nil {
		log.Error("sshClient: session.StdinPipe:", err)
		return err
	}

	if err := c.sess.RequestPty("xterm", int(wdSize.Rows), int(wdSize.Cols), terminalModes); err != nil {
		log.Error("sshClient: session.RequestPty:", err)
		return err
	}
	if err := c.sess.Shell(); err != nil {
		log.Error("sshClient: session.Shell:", err)
		return err
	}

	log.Info("started a login shell on the remote host:", c.addr)
	return err
}

func (c *sshClient) Close() {
	if c.sess != nil {
		c.sess.Close()
	}
	if c.client != nil {
		c.client.Close()
	}
	if c.sessIn != nil {
		c.sessIn.Close()
	}
}

func (c *sshClient) Resize(cols uint16, rows uint16) {
	if c.sess != nil {
		if err := c.sess.WindowChange(int(rows), int(cols)); err != nil {
			log.Error("sshClient: session.WindowChange:", err)
		}
	}
}

func initializePluginClients(serverConfig MCPServerConfig) (*client.Client, error) {
	log.Info("sshClient: initializePluginClients")

	var err error

	// Initialiaze the MCP client
	mcpClient, err := client.NewStdioMCPClient(serverConfig.Command, serverConfig.Env, serverConfig.Args...)
	if err != nil {
		log.Error("sshClient: client.NewInProcessClient:", err)
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel = cancel

	defer func() {
		//cancel()
		//if err := mcpClient.Close(); err != nil {
		//	log.Error("sshClient: mcpClient.Close:", err)
		//}
	}()
	done := make(chan error, 1)

	go func() {
		//mcpClient.Start(ctx)
		_, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Error("sshClient: mcpClient init error:", err)
			return nil, err
		}
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("initialization timed out")
	}

	return mcpClient, nil
}

// [Ab]using the LLM MCP client to call the SSH endpoint "tool" plugin since it implements already
// a "standard" interface to call plugins and we don't want to implement a new one.
func callGetAllNodesPlugin() ([]SshCluster, error) {
	ctx := context.Background()

	var params = make(map[string]any)

	request := mcp.CallToolRequest{}
	request.Params.Name = "get_all_nodes"
	request.Params.Arguments = params

	res, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		log.Error("sshClient: mcpClient.CallTool:", err)
		return nil, err
	}

	if len(res.Content) == 0 {
		return nil, fmt.Errorf("function get_all_nodes failed")
	}

	data, _ := json.Marshal(res)

	textContent := struct {
		Content []mcp.TextContent `json:"content"`
	}{}

	err = json.Unmarshal(data, &textContent)
	if textContent.Content[0].Type != "text" || res.IsError {
		log.Error("sshClient: mcpClient.CallTool:", err)
		return nil, fmt.Errorf("function get_all_nodes failed: %s", textContent.Content[0].Text)
	}

	var clusters []SshCluster
	err = json.Unmarshal([]byte(textContent.Content[0].Text), &clusters)
	if err != nil {
		log.Error("sshClient: json.Unmarshal:", err)
		return nil, err
	}

	return clusters, nil
}

// [Ab]using the LLM MCP client to call the SSH endpoint "tool" plugin since it implements already
// a "standard" interface to call plugins and we don't want to implement a new one.
func callSshEndpointPlugin(sshTarget string) (*SshEndpointData, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		//if err := mcpClient.Close(); err != nil {
		//	log.Error("sshClient: mcpClient.Close:", err)
		//}
	}()

	var params = make(map[string]any)
	params["target"] = sshTarget

	request := mcp.CallToolRequest{}
	request.Params.Name = "get_ssh_connection_info"
	request.Params.Arguments = params

	res, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		log.Error("sshClient: mcpClient.CallTool:", err)
		return nil, err
	}

	if res.IsError || len(res.Content) == 0 {
		return nil, fmt.Errorf("function get_ssh_connection_info failed")
	}

	data, _ := json.Marshal(res)

	textContent := struct {
		Content []mcp.TextContent `json:"content"`
	}{}

	err = json.Unmarshal(data, &textContent)
	if textContent.Content[0].Type != "text" {
		log.Error("sshClient: mcpClient.CallTool:", err)
		return nil, fmt.Errorf("function get_ssh_connection_info failed")
	}

	var sshEndpoint SshEndpointData
	err = json.Unmarshal([]byte(textContent.Content[0].Text), &sshEndpoint)
	if err != nil {
		log.Error("sshClient: json.Unmarshal:", err)
		return nil, err
	}

	return &sshEndpoint, nil
}
