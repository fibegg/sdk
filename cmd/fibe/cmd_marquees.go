package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func marqueesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marquees",
		Aliases: []string{"mq"},
		Short:   "Manage marquees (compute servers)",
		Long: `Manage Fibe marquees — compute infrastructure servers.

A marquee is a server (VPS, bare metal, etc.) that hosts your playgrounds.
Marquees are connected via SSH and managed by Fibe.

SUBCOMMANDS:
  list                  List all marquees
  get <id>              Show marquee details
  create                Create a new marquee
  update <id>           Update marquee settings
  delete <id>           Delete a marquee
  generate-ssh-key <id> Generate SSH key pair
  test-connection <id>  Test SSH connection
  autoconnect-token     Generate autoconnect token`,
	}

	cmd.AddCommand(mqListCmd(), mqGetCmd(), mqCreateCmd(), mqUpdateCmd(), mqDeleteCmd(), mqSSHKeyCmd(), mqTestCmd(), mqAutoconnectCmd())
	return cmd
}

func mqListCmd() *cobra.Command {
	var query, status, name, sort, createdAfter, createdBefore string
	cmd := &cobra.Command{
		Use: "list", Short: "List all marquees",
		Long: `List all marquees accessible to the authenticated user.

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
  fibe marquees list
  fibe mq list -q "prod" --status active
  fibe mq list --sort name_asc -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.MarqueeListParams{}
			if query != "" { params.Q = query }
			if status != "" { params.Status = status }
			if name != "" { params.Name = name }
			if createdAfter != "" { params.CreatedAfter = createdAfter }
			if createdBefore != "" { params.CreatedBefore = createdBefore }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			mqs, err := c.Marquees.List(ctx(), params)
			if err != nil { return err }
			if effectiveOutput() != "table" { outputJSON(mqs); return nil }
			headers := []string{"ID", "NAME", "HOST", "STATUS"}
			rows := make([][]string, len(mqs.Data))
			for i, m := range mqs.Data { rows[i] = []string{fmtInt64(m.ID), m.Name, m.Host, m.Status} }
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
		Use: "get <id>", Short: "Show marquee details", Args: cobra.ExactArgs(1),
		Long: "Get detailed information about a marquee.\n\nEXAMPLES:\n  fibe marquees get 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			mq, err := c.Marquees.Get(ctx(), id)
			if err != nil { return err }
			if effectiveOutput() != "table" { outputJSON(mq); return nil }
			fmt.Printf("ID:     %d\nName:   %s\nHost:   %s:%d\nUser:   %s\nStatus: %s\n", mq.ID, mq.Name, mq.Host, mq.Port, mq.User, mq.Status)
			return nil
		},
	}
}

func mqCreateCmd() *cobra.Command {
	var name, host, user, sshKey, status, dnsProvider string
	var port int
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new marquee",
		Long: "Create a new marquee (compute server) for hosting playgrounds.\n\nHARDWARE CONSTRAINTS:\n  - Marquees represent raw remote Docker hosts via SSH.\n  - Ensure Docker daemon is accessible and SSH Auth is configured securely.\n  - Playgrounds bind strictly to one Marquee.\n\nREQUIRED FLAGS:\n  --name       Marquee name\n  --host       SSH hostname or IP\n  --port       SSH port\n  --user       SSH username\n  --ssh-key    SSH private key content\n\nOPTIONAL FLAGS:\n  --status       Initial status\n  --dns-provider DNS provider name\n\nEXAMPLES:\n  fibe marquees create --name prod --host 10.0.1.5 --port 22 --user deploy --ssh-key \"$(cat ~/.ssh/id_rsa)\"" + generateSchemaDoc(&fibe.MarqueeCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.MarqueeCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") { params.Name = name }
			if cmd.Flags().Changed("host") { params.Host = host }
			if cmd.Flags().Changed("port") { params.Port = port }
			if cmd.Flags().Changed("user") { params.User = user }
			if cmd.Flags().Changed("ssh-key") { params.SSHPrivateKey = sshKey }
			if cmd.Flags().Changed("status") { params.Status = &status }
			if cmd.Flags().Changed("dns-provider") { params.DnsProvider = &dnsProvider }

			if params.Name == "" { return fmt.Errorf("required field 'name' not set") }
			if params.Host == "" { return fmt.Errorf("required field 'host' not set") }
			if params.User == "" { return fmt.Errorf("required field 'user' not set") }
			if params.SSHPrivateKey == "" { return fmt.Errorf("required field 'ssh-key' not set") }

			mq, err := c.Marquees.Create(ctx(), params)
			if err != nil { return err }
			fmt.Printf("Created marquee %d (%s)\n", mq.ID, mq.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name (required)")
	cmd.Flags().StringVar(&host, "host", "", "SSH host (required)")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	cmd.Flags().StringVar(&user, "user", "", "SSH user (required)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH private key (required)")
	cmd.Flags().StringVar(&status, "status", "", "Initial status")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "", "DNS provider name")
	return cmd
}

func mqUpdateCmd() *cobra.Command {
	var name, status, dnsProvider string
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update marquee settings", Args: cobra.ExactArgs(1),
		Long: "Update a marquee's configuration parameters." + generateSchemaDoc(&fibe.MarqueeUpdateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.MarqueeUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") { params.Name = &name }
			if cmd.Flags().Changed("status") { params.Status = &status }
			if cmd.Flags().Changed("dns-provider") { params.DnsProvider = &dnsProvider }
			mq, err := c.Marquees.Update(ctx(), id, params)
			if err != nil { return err }
			fmt.Printf("Updated marquee %d\n", mq.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&status, "status", "", "New status")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "", "DNS provider name")
	return cmd
}

func mqDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a marquee", Args: cobra.ExactArgs(1),
		Long: "Delete a marquee. Cannot delete if active playgrounds exist.\n\nEXAMPLES:\n  fibe marquees delete 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			must(c.Marquees.Delete(ctx(), id))
			fmt.Printf("Marquee %d deleted\n", id)
			return nil
		},
	}
}

func mqSSHKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use: "generate-ssh-key <id>", Short: "Generate SSH key pair for marquee", Args: cobra.ExactArgs(1),
		Long: "Generate a new SSH key pair for a marquee.\nReturns the public key that should be added to the server's authorized_keys.\n\nEXAMPLES:\n  fibe marquees generate-ssh-key 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			result, err := c.Marquees.GenerateSSHKey(ctx(), id)
			if err != nil { return err }
			fmt.Println(result.PublicKey)
			return nil
		},
	}
}

func mqTestCmd() *cobra.Command {
	return &cobra.Command{
		Use: "test-connection <id>", Short: "Test SSH connection to marquee", Args: cobra.ExactArgs(1),
		Long: "Test the SSH connection to a marquee server.\n\nEXAMPLES:\n  fibe marquees test-connection 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			result, err := c.Marquees.TestConnection(ctx(), id)
			if err != nil { return err }
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
		Long: `Generate a short-lived autoconnect token for marquee setup.

The token is valid for 5 minutes and can be used with the connect.sh script.

OPTIONAL FLAGS:
  --email          Email address
  --domain         Domain name
  --ip             IP address
  --ssl-mode       SSL mode
  --dns-provider   DNS provider name

EXAMPLES:
  fibe marquees autoconnect-token --email admin@example.com --domain app.example.com --ip 10.0.1.5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AutoconnectTokenParams{}
			if email != "" { params.Email = email }
			if domain != "" { params.Domain = domain }
			if ip != "" { params.IP = ip }
			if sslMode != "" { params.SSLMode = sslMode }
			if dnsProvider != "" { params.DnsProvider = dnsProvider }
			result, err := c.Marquees.AutoconnectToken(ctx(), params)
			if err != nil { return err }
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
