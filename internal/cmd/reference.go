package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gitwise/internal/knowledge"
	"gitwise/internal/ui"
)

func init() {
	cheat := &cobra.Command{
		Use:     "cheatsheet",
		Aliases: []string{"cheat", "cs"},
		Short:   "Searchable cheatsheet of everyday git commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			printOnly, _ := cmd.Flags().GetBool("print")
			return runCheatsheet(printOnly)
		},
	}
	cheat.Flags().BoolP("print", "p", false, "print the whole cheatsheet instead of opening the browser")

	rootCmd.AddCommand(cheat, &cobra.Command{
		Use:     "glossary [term]",
		Aliases: []string{"define", "g"},
		Short:   "Explain git vocabulary (HEAD, upstream, staging area, ...)",
		RunE:    func(cmd *cobra.Command, args []string) error { return printGlossary(args) },
	}, &cobra.Command{
		Use:   "learn [lesson-id]",
		Short: "Interactive hands-on lessons in a throwaway sandbox repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			return ui.RunLearn(id)
		},
	})
}

type cheatEntry struct {
	cat string
	cmd *knowledge.Command
	ex  knowledge.Example
}

func cheatEntries() []cheatEntry {
	var out []cheatEntry
	for _, cat := range knowledge.Categories {
		for i := range knowledge.Commands {
			c := &knowledge.Commands[i]
			if c.Category != cat {
				continue
			}
			for _, ex := range c.Examples {
				out = append(out, cheatEntry{cat, c, ex})
			}
		}
	}
	return out
}

func runCheatsheet(printOnly bool) error {
	entries := cheatEntries()
	if printOnly || !interactive() {
		for _, cat := range knowledge.Categories {
			var rows [][2]string
			for _, e := range entries {
				if e.cat == cat {
					rows = append(rows, [2]string{e.ex.Cmd, e.ex.Desc})
				}
			}
			fmt.Println("\n" + ui.Heading.Render(cat))
			fmt.Print(ui.CommandTable(rows))
		}
		return nil
	}

	items := make([]ui.Item, len(entries))
	for i, e := range entries {
		items[i] = ui.Item{Name: e.ex.Cmd, Desc: e.cat + " · " + e.ex.Desc, Extra: e.cat + " " + e.ex.Desc}
	}
	i, err := ui.Pick("Git cheatsheet — press / to search", items)
	if err != nil {
		return err
	}
	return explain(splitCommandLine(entries[i].ex.Cmd))
}

func printGlossary(args []string) error {
	terms := knowledge.Glossary
	if q := strings.Join(args, " "); q != "" {
		if terms = knowledge.FindTerms(q); len(terms) == 0 {
			return fmt.Errorf("no glossary entry matches %q — run gitwise glossary to see them all", q)
		}
	}
	w := width()
	for _, t := range terms {
		fmt.Println(ui.Heading.Render(t.Name))
		fmt.Println(ui.Wrap(ui.Inline(t.Def), w-2))
		fmt.Println()
	}
	return nil
}
