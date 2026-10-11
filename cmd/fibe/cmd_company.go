package main

import (
	"encoding/json"
	"fmt"
	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func companyCmd() *cobra.Command {
	var companyID int64
	root := &cobra.Command{Use: "company", Short: "Inspect and recover operations in an explicitly selected Company", Long: "Company IDs are credential-owner expectations, not authority grants. Recovery, cancellation, repository reconciliation/adoption and installation management require the server's live human authority checks. Use a Company-owned human credential/profile."}
	root.PersistentFlags().Int64Var(&companyID, "company-id", 0, "Required Company owner ID (verified through request and response headers)")
	action := func(use, help string, nargs int, call func(*cobra.Command, []string, *fibe.Client) (any, error)) *cobra.Command {
		return &cobra.Command{Use: use, Short: help, Args: cobra.ExactArgs(nargs), RunE: func(cmd *cobra.Command, args []string) error {
			if companyID <= 0 {
				return fmt.Errorf("--company-id must be a positive Company ID")
			}
			out, err := call(cmd, args, newClient())
			if err != nil {
				return err
			}
			if out != nil {
				outputJSON(out)
			}
			return nil
		}}
	}
	principals := &cobra.Command{Use: "principals", Short: "Company service principals and explicit runtime-principal recovery"}
	principals.AddCommand(action("list", "List Company service principals", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.TeamServicePrincipals(ctx(), companyID)
	}))
	var name string
	create := action("create", "Issue a service principal; a live human admin is required", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.CreateTeamServicePrincipal(ctx(), companyID, name)
	})
	create.Flags().StringVar(&name, "name", "", "Service principal name")
	principals.AddCommand(create, action("revoke <id>", "Revoke a Company service principal", 1, func(_ *cobra.Command, args []string, c *fibe.Client) (any, error) {
		id, err := envPackID(args[0])
		if err != nil {
			return nil, err
		}
		return nil, c.RevokeTeamServicePrincipal(ctx(), companyID, id)
	}))
	var selection string
	recover := action("recover <id>", "Recover a revoked runtime principal; identical selections replay, changed selections conflict", 1, func(cmd *cobra.Command, args []string, c *fibe.Client) (any, error) {
		id, err := envPackID(args[0])
		if err != nil {
			return nil, err
		}
		var p struct {
			ReplacementAgentIDs []int64 `json:"replacement_agent_ids"`
		}
		if err = applyFromFile(&p); err != nil {
			return nil, err
		}
		if cmd.Flags().Changed("agent-ids") {
			if err = json.Unmarshal([]byte(selection), &p.ReplacementAgentIDs); err != nil {
				return nil, fmt.Errorf("--agent-ids must be a JSON list of positive integer Agent IDs")
			}
		}
		return c.RecoverRuntimePrincipal(ctx(), companyID, id, p.ReplacementAgentIDs)
	})
	recover.Flags().StringVar(&selection, "agent-ids", "", "Required explicit JSON list, e.g. '[12,34]' or '[]'; alternatively --from-file replacement_agent_ids")
	principals.AddCommand(recover)
	subscriptions := &cobra.Command{Use: "subscriptions", Short: "Durable Company subscription cancellation and current status"}
	subscriptions.AddCommand(action("list", "Read subscription and durable cancellation states", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.CompanySubscriptions(ctx(), companyID)
	}))
	var retry bool
	cancel := action("cancel <id>", "Request cancellation (202 is durable intent); use list for reconciled status", 1, func(_ *cobra.Command, args []string, c *fibe.Client) (any, error) {
		id, err := envPackID(args[0])
		if err != nil {
			return nil, err
		}
		return c.CancelCompanySubscription(ctx(), companyID, id, retry)
	})
	cancel.Flags().BoolVar(&retry, "retry-failed", false, "Explicitly retry a failed cancellation with current human authority")
	subscriptions.AddCommand(cancel)
	operations := &cobra.Command{Use: "repository-operations", Short: "Reconcile exact remote destinations before independent local adoption"}
	operations.AddCommand(action("list", "List durable Company Repository receipts", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.CompanyRepositoryOperations(ctx(), companyID)
	}))
	for _, kind := range []string{"reconcile", "adopt"} {
		var connectionID int64
		operation := kind
		command := action(operation+" <request-id>", "Requires a live human and current installation authority for the exact Company destination", 1, func(cmd *cobra.Command, args []string, c *fibe.Client) (any, error) {
			var connection *int64
			if cmd.Flags().Changed("connection-id") {
				connection = &connectionID
			}
			if operation == "reconcile" {
				return c.ReconcileCompanyRepositoryOperation(ctx(), companyID, args[0], connection)
			}
			return c.AdoptCompanyRepositoryOperation(ctx(), companyID, args[0], connection)
		})
		command.Flags().Int64Var(&connectionID, "connection-id", 0, "Optional current Company installation connection ID")
		operations.AddCommand(command)
	}
	installations := &cobra.Command{Use: "git-installations", Short: "Current Company organization installations; no token export"}
	installations.AddCommand(action("list", "Read live availability and verified organization authority", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.CompanyGitInstallations(ctx(), companyID)
	}))
	var installationID, verificationID int64
	connect := action("connect", "Verify and connect an organization installation using the live human's verification connection", 0, func(_ *cobra.Command, _ []string, c *fibe.Client) (any, error) {
		return c.ConnectCompanyGitInstallation(ctx(), companyID, installationID, verificationID)
	})
	connect.Flags().Int64Var(&installationID, "installation-id", 0, "GitHub organization installation ID")
	connect.Flags().Int64Var(&verificationID, "verification-connection-id", 0, "Existing human GitHub verification connection ID")
	installations.AddCommand(connect, action("remove <id>", "Remove a current Company installation; pending operations must be reconciled first", 1, func(_ *cobra.Command, args []string, c *fibe.Client) (any, error) {
		id, err := envPackID(args[0])
		if err != nil {
			return nil, err
		}
		return nil, c.RemoveCompanyGitInstallation(ctx(), companyID, id)
	}))
	root.AddCommand(principals, subscriptions, operations, installations)
	return root
}
