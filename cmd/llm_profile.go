package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
)

var (
	profileName        string
	profileKey         string
	profileModel       string
	profileProvider    string
	profileDelete      string
	profileList        bool
	profileInteractive bool
)

// llmCmd manages encrypted LLM profiles (stored under output/config).
var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage encrypted LLM profiles (provider/model/key)",
	Run: func(cmd *cobra.Command, args []string) {
		interactiveRequested := profileInteractive || (profileName == "" && profileDelete == "" && !profileList)
		if interactiveRequested {
			runInteractiveProfile()
			return
		}
		secret := strings.TrimSpace(profileName)
		if secret == "" {
			utils.Info("please set --name for list/save/delete")
			return
		}
		path := utils.DefaultLLMProfilePath("")
		profiles, err := utils.LoadLLMProfiles(path, secret)
		if err != nil && !os.IsNotExist(err) {
			utils.Warning("load profiles failed (use profile name as secret): %v", err)
		}

		if profileList {
			if len(profiles) == 0 {
				utils.Info("No LLM profiles saved.")
				return
			}
			for _, p := range profiles {
				utils.Info("name=%s provider=%s model=%s", p.Name, p.Provider, p.Model)
			}
			return
		}

		if profileDelete != "" {
			profiles = utils.DeleteProfile(profiles, profileDelete)
			if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
				utils.Error("delete profile failed: %v", err)
				return
			}
			utils.Success("Profile %s deleted", profileDelete)
			return
		}

		if profileName != "" {
			p := utils.LLMProfile{
				Name:     profileName,
				Provider: firstNonEmpty(profileProvider, utils.DefaultLLMProvider),
				Model:    firstNonEmpty(profileModel, utils.DefaultLLMModel),
				APIKey:   strings.TrimSpace(profileKey),
			}
			if p.APIKey == "" {
				utils.Error("profile key is empty, use --key")
				return
			}
			profiles = utils.UpsertProfile(profiles, p)
			if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
				utils.Error("save profile failed: %v", err)
				return
			}
			utils.Success("Profile %s saved (path=%s)", p.Name, path)
			return
		}

		utils.Info("No action taken. Use --list, --name, --delete, or run with -h for help.")
	},
}

func init() {
	rootCmd.AddCommand(llmCmd)
	llmCmd.Flags().StringVar(&profileName, "name", "", "Profile name (also used as encryption secret)")
	llmCmd.Flags().StringVar(&profileKey, "key", "", "API key for profile (required for save)")
	llmCmd.Flags().StringVar(&profileModel, "model", "", "Model id, e.g. gemini-3-flash-preview")
	llmCmd.Flags().StringVar(&profileProvider, "provider", "", "LLM provider, default google")
	llmCmd.Flags().BoolVar(&profileList, "list", false, "List saved profiles")
	llmCmd.Flags().StringVar(&profileDelete, "delete", "", "Delete profile by name")
	llmCmd.Flags().BoolVarP(&profileInteractive, "interactive", "i", false, "Interactive mode: prompt for name/model/key")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func runInteractiveProfile() {
	reader := bufio.NewReader(os.Stdin)
	name := strings.TrimSpace(prompt(reader, "Profile name (required): ", ""))
	if name == "" {
		utils.Error("Profile name is required")
		return
	}
	secret := name
	path := utils.DefaultLLMProfilePath("")
	profiles, _ := utils.LoadLLMProfiles(path, secret)
	defaultProvider := utils.DefaultLLMProvider
	defaultModel := utils.DefaultLLMModel
	defaultKey := ""
	provider := strings.TrimSpace(prompt(reader, fmt.Sprintf("Provider [%s]: ", defaultProvider), defaultProvider))
	model := strings.TrimSpace(promptModel(reader, defaultModel))
	key := strings.TrimSpace(prompt(reader, "API key (leave blank to keep existing): ", defaultKey))
	if key == "" {
		utils.Error("API key is required")
		return
	}
	p := utils.LLMProfile{
		Name:     name,
		Provider: provider,
		Model:    model,
		APIKey:   key,
	}
	profiles = utils.UpsertProfile(profiles, p)
	if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
		utils.Error("save profile failed: %v", err)
		return
	}
	utils.Success("Profile %s saved (path=%s)", p.Name, path)
}

func prompt(r *bufio.Reader, msg, def string) string {
	fmt.Print(msg)
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func promptModel(r *bufio.Reader, def string) string {
	options := []string{"gemini-2.5-flash", "gemini-3-flash-preview"}
	fmt.Printf("Model options:\n")
	for i, opt := range options {
		fmt.Printf(" %d) %s\n", i+1, opt)
	}
	fmt.Printf("Choose model [default %s]: ", def)
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	switch text {
	case "1":
		return options[0]
	case "2":
		return options[1]
	case "":
		return def
	default:
		return text
	}
}

func getOrPromptSecret(allowPrompt bool) string {
	return strings.TrimSpace(profileName)
}
