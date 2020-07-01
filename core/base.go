package core

// Config 结构体
type Config struct {
	RunType    string   `json:"run_type"`
	LocalAddr  string   `json:"local_addr"`
	LocalPort  int      `json:"local_port"`
	RemoteAddr string   `json:"remote_addr"`
	RemotePort int      `json:"remote_port"`
	Password   []string `json:"password"`
	LogLevel   int      `json:"log_level"`
}

// SSL 结构体
type SSL struct {
	Cert          string   `json:"cert"`
	Cipher        string   `json:"cipher"`
	CipherTls13   string   `json:"cipher_tls13"`
	Alpn          []string `json:"alpn"`
	ReuseSession  bool     `json:"reuse_session"`
	SessionTicket bool     `json:"session_ticket"`
	Curves        string   `json:"curves"`
	Sni           string   `json:"sni"`
}

// TCP 结构体
type TCP struct {
	NoDelay      bool `json:"no_delay"`
	KeepAlive    bool `json:"keep_alive"`
	ReusePort    bool `json:"reuse_port"`
	FastOpen     bool `json:"fast_open"`
	FastOpenQlen int  `json:"fast_open_qlen"`
}
