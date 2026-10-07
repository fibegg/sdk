package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func hostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hosts",
		Aliases: []string{"host"},
		Short:   "Manage hosts (compute servers)",
		Long: `Manage Fibe hosts: compute infrastructure servers.

A host is a server (VPS, bare metal, etc.) that hosts your playgrounds.
Hosts are connected via SSH and managed by Fibe.
Actions on unpaid Hosts fail with HOST_NOT_FUNDED; billing/funding
and cleanup remain allowed.

SUBCOMMANDS:
  list                  List all hosts
  get <id-or-name>      Show host details
  create                Create a new host
  update <id-or-name>           Update host settings
  delete <id-or-name>           Delete a host
  generate-ssh-key <id-or-name> Generate SSH key pair
  test-connection <id-or-name>  Test SSH connection
  autoconnect-token     Generate autoconnect token`,
	}

	cmd.AddCommand(mqListCmd(), mqGetCmd(), mqCreateCmd(), mqUpdateCmd(), mqDeleteCmd(), mqSSHKeyCmd(), mqTestCmd(), mqAutoconnectCmd())
	return cmd
}

func mqListCmd() *cobra.Command {
	var query, status, name, sort, createdAfter, createdBefore string
	cmd := &cobra.Command{
		Use: "list", Short: "List all hosts",
		Long: `List all hosts accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name, host (substring match)
  --status              Filter by exact status. Values: active, inactive
  --name                Filter by name (substring match)

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, NAME, HOST, STATUS
  Use --output json for full details.

EXAMPLES:
  fibe hosts list
  fibe mq list -q "prod" --status active
  fibe mq list --sort name_asc -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.HostListParams{}
			if query != "" {
				params.Q = query
			}
			if status != "" {
				params.Status = status
			}
			if name != "" {
				params.Name = name
			}
			if createdAfter != "" {
				params.CreatedAfter = createdAfter
			}
			if createdBefore != "" {
				params.CreatedBefore = createdBefore
			}
			if sort != "" {
				params.Sort = sort
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			mqs, err := c.Hosts.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(mqs)
				return nil
			}
			headers := []string{"ID", "NAME", "HOST", "STATUS"}
			rows := make([][]string, len(mqs.Data))
			for i, m := range mqs.Data {
				rows[i] = []string{fmtInt64(m.ID), m.Name, m.Host, m.Status}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name, host")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func mqGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id-or-name>", Short: "Show host details", Args: cobra.ExactArgs(1),
		Long: "Get detailed information about a host.\n\nEXAMPLES:\n  fibe hosts get 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			mq, err := c.Hosts.GetByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(mq)
				return nil
			}
			fmt.Printf("ID:     %d\nName:   %s\nHost:   %s:%d\nUser:   %s\nStatus: %s\n", mq.ID, mq.Name, mq.Host, mq.Port, mq.User, mq.Status)
			return nil
		},
	}
}

func mqCreateCmd() *cobra.Command {
	var name, host, user, sshKey, status, dnsProvider, tlsCertificateSource, tlsCertificatePEM, tlsPrivateKeyPEM string
	var port int
	var httpsEnabled bool
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new host",
		Long: "Create a new host (compute server) for hosting playgrounds.\n\nHARDWARE CONSTRAINTS:\n  - Hosts represent raw remote Docker hosts via SSH.\n  - Ensure Docker daemon is accessible and SSH Auth is configured securely.\n  - Playgrounds bind strictly to one Host.\n\nREQUIRED FLAGS:\n  --name       Host name\n  --host       SSH hostname or IP\n  --port       SSH port\n  --user       SSH username\n  --ssh-key    SSH private key content\n\nOPTIONAL FLAGS:\n  --status       Initial status\n  --dns-provider DNS provider name\n\nEXAMPLES:\n  fibe hosts create --name prod --host 10.0.1.5 --port 22 --user deploy --ssh-key \"$(cat ~/.ssh/id_rsa)\"" + generateSchemaDoc(&fibe.HostCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.HostCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("host") {
				params.Host = host
			}
			if cmd.Flags().Changed("port") {
				params.Port = port
			}
			if cmd.Flags().Changed("user") {
				params.User = user
			}
			if cmd.Flags().Changed("ssh-key") {
				params.SSHPrivateKey = sshKey
			}
			if cmd.Flags().Changed("status") {
				params.Status = &status
			}
			if cmd.Flags().Changed("https-enabled") {
				params.HttpsEnabled = &httpsEnabled
			}
			if cmd.Flags().Changed("tls-certificate-source") {
				params.TlsCertificateSource = &tlsCertificateSource
			}
			if cmd.Flags().Changed("tls-certificate-pem") {
				params.TlsCertificatePEM = &tlsCertificatePEM
			}
			if cmd.Flags().Changed("tls-private-key-pem") {
				params.TlsPrivateKeyPEM = &tlsPrivateKeyPEM
			}
			if cmd.Flags().Changed("dns-provider") {
				params.DnsProvider = &dnsProvider
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.Host == "" {
				return fmt.Errorf("required field 'host' not set")
			}
			if params.User == "" {
				return fmt.Errorf("required field 'user' not set")
			}
			if params.SSHPrivateKey == "" {
				return fmt.Errorf("required field 'ssh-key' not set")
			}

			mq, err := c.Hosts.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(mq)
				return nil
			}
			fmt.Printf("Created host %d (%s)\n", mq.ID, mq.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name (required)")
	cmd.Flags().StringVar(&host, "host", "", "SSH host (required)")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	cmd.Flags().StringVar(&user, "user", "", "SSH user (required)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH private key (required)")
	cmd.Flags().StringVar(&status, "status", "", "Initial status")
	cmd.Flags().BoolVar(&httpsEnabled, "https-enabled", true, "Enable HTTPS routing")
	cmd.Flags().StringVar(&tlsCertificateSource, "tls-certificate-source", "", "TLS certificate source: automatic or provided")
	cmd.Flags().StringVar(&tlsCertificatePEM, "tls-certificate-pem", "", "Provided TLS certificate PEM")
	cmd.Flags().StringVar(&tlsPrivateKeyPEM, "tls-private-key-pem", "", "Provided TLS private key PEM")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "", "DNS provider name")
	return cmd
}

func mqUpdateCmd() *cobra.Command {
	var name, status, dnsProvider, tlsCertificateSource, tlsCertificatePEM, tlsPrivateKeyPEM string
	var httpsEnabled bool
	cmd := &cobra.Command{
		Use: "update <id-or-name>", Short: "Update host settings", Args: cobra.ExactArgs(1),
		Long: "Update a host's configuration parameters." + generateSchemaDoc(&fibe.HostUpdateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.HostUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("status") {
				params.Status = &status
			}
			if cmd.Flags().Changed("https-enabled") {
				params.HttpsEnabled = &httpsEnabled
			}
			if cmd.Flags().Changed("tls-certificate-source") {
				params.TlsCertificateSource = &tlsCertificateSource
			}
			if cmd.Flags().Changed("tls-certificate-pem") {
				params.TlsCertificatePEM = &tlsCertificatePEM
			}
			if cmd.Flags().Changed("tls-private-key-pem") {
				params.TlsPrivateKeyPEM = &tlsPrivateKeyPEM
			}
			if cmd.Flags().Changed("dns-provider") {
				params.DnsProvider = &dnsProvider
			}
			mq, err := c.Hosts.UpdateByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(mq)
				return nil
			}
			fmt.Printf("Updated host %d\n", mq.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&status, "status", "", "New status")
	cmd.Flags().BoolVar(&httpsEnabled, "https-enabled", true, "Enable HTTPS routing")
	cmd.Flags().StringVar(&tlsCertificateSource, "tls-certificate-source", "", "TLS certificate source: automatic or provided")
	cmd.Flags().StringVar(&tlsCertificatePEM, "tls-certificate-pem", "", "Provided TLS certificate PEM")
	cmd.Flags().StringVar(&tlsPrivateKeyPEM, "tls-private-key-pem", "", "Provided TLS private key PEM")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "", "DNS provider name")
	return cmd
}

func mqDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id-or-name>", Short: "Delete a host", Args: cobra.ExactArgs(1),
		Long: "Delete a host. Cannot delete if active playgrounds exist.\n\nEXAMPLES:\n  fibe hosts delete 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Hosts.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Host %s deleted\n", args[0])
			return nil
		},
	}
}

func mqSSHKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use: "generate-ssh-key <id-or-name>", Short: "Generate SSH key pair for host", Args: cobra.ExactArgs(1),
		Long: "Generate a new SSH key pair for a host.\nReturns the public key that should be added to the server's authorized_keys.\n\nEXAMPLES:\n  fibe hosts generate-ssh-key 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			progress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{})
			progress.Start("generating SSH key for host " + args[0] + "...")
			defer progress.Stop()
			c := newClient(fibe.WithProgress(progress.Progress("generating SSH key for host " + args[0])))
			result, err := c.Hosts.GenerateSSHKeyByIdentifier(ctx(), args[0])
			progress.Stop()
			if err != nil {
				return err
			}
			fmt.Println(result.PublicKey)
			return nil
		},
	}
}

func mqTestCmd() *cobra.Command {
	return &cobra.Command{
		Use: "test-connection <id-or-name>", Short: "Test SSH connection to host", Args: cobra.ExactArgs(1),
		Long: "Test the SSH connection to a host server.\n\nEXAMPLES:\n  fibe hosts test-connection 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			progress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{})
			progress.Start("testing connection to host " + args[0] + "...")
			defer progress.Stop()
			c := newClient(fibe.WithProgress(progress.Progress("testing connection to host " + args[0])))
			result, err := c.Hosts.TestConnectionByIdentifier(ctx(), args[0])
			progress.Stop()
			if err != nil {
				return err
			}
			if result.Success {
				fmt.Println("Connection successful")
			} else {
				fmt.Printf("Connection failed: %s\n", result.Error)
			}
			return nil
		},
	}
}

func mqAutoconnectCmd() *cobra.Command {
	var email, domain, ip, sslMode, dnsProvider string
	cmd := &cobra.Command{
		Use: "autoconnect-token", Short: "Generate autoconnect token",
		Long: `Generate a short-lived autoconnect token for host setup.

The token is valid for 5 minutes and can be used with the connect.sh script.

OPTIONAL FLAGS:
  --email          Email address
  --domain         Domain name
  --ip             IP address
  --ssl-mode       SSL mode
  --dns-provider   DNS provider name

EXAMPLES:
  fibe hosts autoconnect-token --email admin@example.com --domain app.example.com --ip 10.0.1.5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AutoconnectTokenParams{}
			if email != "" {
				params.Email = email
			}
			if domain != "" {
				params.Domain = domain
			}
			if ip != "" {
				params.IP = ip
			}
			if sslMode != "" {
				params.SSLMode = sslMode
			}
			if dnsProvider != "" {
				params.DnsProvider = dnsProvider
			}
			result, err := c.Hosts.AutoconnectToken(ctx(), params)
			if err != nil {
				return err
			}
			fmt.Println(result.Token)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain name")
	cmd.Flags().StringVar(&ip, "ip", "", "IP address")
	cmd.Flags().StringVar(&sslMode, "ssl-mode", "", "SSL mode")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "", "DNS provider name")
	return cmd
}
