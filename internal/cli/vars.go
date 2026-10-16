package cli

import (
	"fmt"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/SandSoftwareSolutions/Foundry/internal/project"
	"github.com/SandSoftwareSolutions/Foundry/internal/render"
	"github.com/SandSoftwareSolutions/Foundry/internal/template"
)

// asker asks the user for the value of a placeholder. A nil asker means "not allowed to ask" (no terminal,
// --non-interactive, or machine readable output).
type asker func(name, prompt, def string) (string, error)

// surveyAsk asks on the terminal.
func surveyAsk(name, prompt, def string) (string, error) {
	msg := prompt
	if msg == "" {
		msg = name + ":"
	}
	var answer string
	if err := survey.AskOne(&survey.Input{Message: msg, Default: def}, &answer); err != nil {
		return "", fmt.Errorf("cancelled")
	}
	return answer, nil
}

// fillDeclared gives every placeholder a template declares a value: the one from --var if there is one, else
// the answer of the user, else (when asking is not allowed) the declared default. A declared placeholder with
// neither stays unset and is reported later like any other missing one.
func fillDeclared(specs []template.VarSpec, have render.Vars, ask asker) (render.Vars, error) {
	out := render.Vars{}
	for k, v := range have {
		out[k] = v
	}
	for _, spec := range specs {
		if !render.ValidKey(spec.Name) {
			return nil, fmt.Errorf("the template declares a placeholder with an invalid name %q", spec.Name)
		}
		if _, set := out[spec.Name]; set {
			continue
		}
		switch {
		case ask != nil:
			v, err := ask(spec.Name, spec.Prompt, spec.Default)
			if err != nil {
				return nil, err
			}
			out[spec.Name] = v
		case spec.Default != "":
			out[spec.Name] = spec.Default
		}
	}
	return out, nil
}

// unresolvedNames lists the placeholder names a plan could not fill, sorted and without duplicates.
func unresolvedNames(plan *project.Plan) []string {
	set := map[string]bool{}
	for _, names := range plan.Unresolved {
		for _, n := range names {
			set[n] = true
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// askMissing asks for each of the given placeholders.
func askMissing(names []string, have render.Vars, ask asker) (render.Vars, error) {
	out := render.Vars{}
	for k, v := range have {
		out[k] = v
	}
	for _, n := range names {
		v, err := ask(n, n+":", "")
		if err != nil {
			return nil, err
		}
		out[n] = v
	}
	return out, nil
}

// strictError explains which placeholders would have stayed empty.
func strictError(plan *project.Plan) error {
	return fmt.Errorf("placeholders without a value (--strict): %s\n  Set them with --var NAME=value; \"foundry template vars <template>\" lists what a template uses.\n  %s",
		strings.Join(unresolvedNames(plan), ", "), strings.Join(plan.UnresolvedSummary(), "\n  "))
}
