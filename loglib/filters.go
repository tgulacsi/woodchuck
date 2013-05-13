package loglib

import (
	"errors"
	"log"
	"regexp"
	"strings"
)

type Matcher interface {
	Match(m *Message) bool
}

type reFilter struct {
	Field string
	Re    *regexp.Regexp
}

func (f reFilter) Match(m *Message) (b bool) {
	var v string
	switch f.Field {
	case "host":
		v = m.Host
	case "facility":
		v = m.Facility
	}
	b = f.Re.MatchString(v)
	log.Printf("M %s=%s ?%s: %b", f.Field, v, f.Re, b)
	return
}

type rangeFilter struct {
	Field     string
	sign      int8
	Threshold int64
}

func (f rangeFilter) Match(m *Message) (b bool) {
	var v int64
	switch f.Field {
	case "level":
		v = int64(m.Level)
	}
	if f.sign > 0 {
		b = v > f.Threshold
	} else if f.sign < 0 {
		b = v < f.Threshold
	}
	b = v == f.Threshold
	log.Printf("M %s=%d ?%d %d: %b", f.Field, v, f.sign, f.Threshold, b)
	return
}

type ConfigTree interface {
	// Get the value at key in the TomlTree. Key is a dot-separated path (e.g. a.b.c). Returns nil if the path does not exist in the tree.
	Get(key string) interface{}
	// Keys returns the keys of the toplevel tree. Warning: this is a costly operation.
	Keys() []string
}

func BuildMatchers(tree ConfigTree) (matchers map[string]Matcher, err error) {
	tree = getSubtree(tree, "filters")
	keys := tree.Keys()
	log.Printf("keys of %s: %s", tree, keys)
	if 0 == len(keys) {
		return nil, nil
	}
	matchers = make(map[string]Matcher, len(keys))
	var (
		sub   ConfigTree
		sign  int8
		field string
	)
	for _, k := range keys {
		sub = tree.Get(k).(ConfigTree)
		field = sub.Keys()[0]
		switch x := sub.Get(field).(type) {
		case string:
			matchers[k] = reFilter{Field: field, Re: regexp.MustCompile(x)}
		case int64:
			switch field[len(field)-3:] {
			case "_lt":
				sign = -1
				field = field[:len(field)-3]
			case "gt":
				sign = 1
				field = field[:len(field)-3]
			default:
				sign = 0
			}
			matchers[k] = rangeFilter{Field: field, sign: sign, Threshold: x}
		}
	}
	return
}

func getSubtree(tree ConfigTree, name string) ConfigTree {
	if tree.Get(name) != nil {
		return tree.Get(name).(ConfigTree)
	}
	return tree
}
func getList(tree ConfigTree, name string) (arr []string) {
	v := tree.Get(name)
	if v == nil {
		return
	}
	switch x := v.(type) {
	case []string:
		return x
	case string:
		return []string{x}
	case []interface{}:
		var i int
		arr = make([]string, len(x))
		for i, v = range x {
			arr[i] = v.(string)
		}
		return
	default:
		log.Fatalf("getList(%s, %s) = %s (%T)", tree, name, v, v)
	}
	return
}

type Alerter interface {
	Send(*Message, SMSSender, EmailSender) error
}

type emailAlert struct {
	To []string
}

func (a emailAlert) Send(m *Message, sms SMSSender, email EmailSender) error {
	return email.Send(a.To, m.String(), []byte(m.Long()))
}

type smsAlert struct {
	To []string
}

func (a smsAlert) Send(m *Message, sms SMSSender, email EmailSender) error {
	var err error
	errs := make([]string, 0, len(a.To))
	for _, to := range a.To {
		if err = sms.Send(to, m.String()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}

type mantisAlert struct {
	Url string
}

func (a mantisAlert) Send(m *Message, sms SMSSender, email EmailSender) error {
	log.Printf("unimplemented alert mantis")
	return nil
}

func BuildAlerters(tree ConfigTree) (destinations map[string]Alerter, err error) {
	tree = getSubtree(tree, "destinations")
	keys := tree.Keys()
	log.Printf("keys of %s: %s", tree, keys)
	if 0 == len(keys) {
		return nil, nil
	}
	destinations = make(map[string]Alerter, len(keys))
	var (
		sub ConfigTree
		to  []string = make([]string, 0, 1)
		v   interface{}
	)

	for _, k := range keys {
		sub = tree.Get(k).(ConfigTree)
		if to = getList(sub, "email"); to != nil {
			destinations[k] = emailAlert{To: to}
			continue
		}
		if to = getList(sub, "sms"); to != nil {
			destinations[k] = smsAlert{To: to}
			continue
		}
		v = sub.Get("mantis")
		destinations[k] = mantisAlert{Url: v.(string)}
	}
	return
}

type Rule struct {
	Name string
	If   []Matcher
	Then []Alerter
}

// AND-match
func (rul Rule) Match(m *Message) bool {
	if len(rul.If) == 0 {
		return false
	}
	for _, mr := range rul.If {
		if !mr.Match(m) {
			return false
		}
	}
	return true
}

func (rul Rule) Do(m *Message, sms SMSSender, email EmailSender) (err error) {
	if len(rul.Then) == 0 {
		return
	}
	errs := make([]string, 0, len(rul.Then))
	for _, al := range rul.Then {
		if err = al.Send(m, sms, email); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}

func BuildRules(tree ConfigTree, matchers map[string]Matcher, alerters map[string]Alerter) (rules []Rule, err error) {
	tree = getSubtree(tree, "rules")
	keys := tree.Keys()
	log.Printf("keys of %s: %s", tree, keys)
	if 0 == len(keys) {
		return nil, nil
	}
	var (
		sub     ConfigTree
		subkeys []string
	)
	rules = make([]Rule, 0, len(keys))
	for _, nm := range keys {
		sub = tree.Get(nm).(ConfigTree)
		subkeys = getList(sub, "if")
		ifs := make([]Matcher, len(subkeys))
		for i, k := range subkeys {
			ifs[i] = matchers[k]
		}
		subkeys = getList(sub, "then")
		thens := make([]Alerter, len(subkeys))
		for i, k := range subkeys {
			thens[i] = alerters[k]
		}
		rules = append(rules, Rule{Name: nm, If: ifs, Then: thens})
		log.Printf("%v => %v", sub, rules[len(rules)-1])
	}
	return
}
