package ui

import "github.com/egoist/mygo/internal/platform"

// The on-screen keyboard of a text input on phones and tablets, in upstream
// #940's API (FORK.md, "向 #940 收敛"): Keyboard, ReturnKey, Capitalize,
// Autocorrect and Autofill. Until the fork takes #940's backends they reach
// the fork's as platform.TextInputOptions. Hardware keyboards and desktops
// type as they do.

// KeyboardType is the layout of the on-screen keyboard for a text input.
type KeyboardType uint8

// The keyboards of phones and tablets.
const (
	KeyboardDefault KeyboardType = iota
	KeyboardEmail
	KeyboardNumber
	KeyboardDecimal
	KeyboardPhone
	KeyboardURL
	KeyboardSearch
	// KeyboardASCII is the fork's: letters of ASCII only, as for a host
	// name. #940 has no counterpart.
	KeyboardASCII
)

var keyboardNames = [...]string{"", "email", "number", "decimal", "phone", "url", "search", "ascii"}

// ReturnKeyType is what the return key of an on-screen keyboard says.
type ReturnKeyType uint8

// The return keys of phones and tablets. ReturnNext moves the focus to the
// next element after it submits.
const (
	ReturnDefault ReturnKeyType = iota
	ReturnDone
	ReturnGo
	ReturnNext
	ReturnSearch
	ReturnSend
)

var returnNames = [...]string{"", "done", "go", "next", "search", "send"}

// Capitalization is what an on-screen keyboard capitalizes as one types.
type Capitalization uint8

// The capitalizations of on-screen keyboards.
const (
	CapitalizeDefault Capitalization = iota
	CapitalizeNone
	CapitalizeWords
	CapitalizeSentences
	CapitalizeCharacters
)

var capitalizeNames = [...]string{"", "none", "words", "sentences", "all"}

// AutofillKind names what a text input holds, for the system to offer it
// from its passwords and contacts. Password() is still what hides a
// password's text.
type AutofillKind uint8

// The kinds of text the system fills in.
const (
	AutofillNone AutofillKind = iota
	AutofillUsername
	AutofillPassword
	AutofillNewPassword
	AutofillOneTimeCode
	AutofillEmail
	AutofillName
	AutofillPhone
	AutofillAddress
	AutofillPostalCode
	AutofillCreditCard
)

var autofillNames = [...]string{"", "username", "password", "newPassword", "oneTimeCode", "emailAddress", "name", "telephoneNumber", "fullStreetAddress", "postalCode", "creditCardNumber"}

// KeyboardDismissMode chooses how scrolling dismisses the on-screen
// keyboard. It is the fork's (#940 has no counterpart); iOS implements
// Interactive, and other platforms treat it as none.
type KeyboardDismissMode string

const (
	KeyboardDismissNone        KeyboardDismissMode = ""
	KeyboardDismissOnDrag      KeyboardDismissMode = "on-drag"
	KeyboardDismissInteractive KeyboardDismissMode = "interactive"
)

// inputTraits are this frame's keyboard traits of an editable field or of a
// custom HandleInput/TextCaret element, such as a terminal; a frame that
// sets none restores the defaults.
type inputTraits struct {
	Dismiss, Keyboard, Return, Content, Correction, Capitalization string
}

func (o inputTraits) nativeOptions() platform.TextInputOptions {
	return platform.TextInputOptions{
		Keyboard: o.Keyboard, Return: o.Return, Content: o.Content,
		Correction: o.Correction, Capitalization: o.Capitalization, Dismiss: o.Dismiss,
	}
}

// name returns names[i], or "" (the system's default) for a value it does
// not know.
func name[T ~uint8](names []string, i T) string {
	if int(i) < len(names) {
		return names[i]
	}
	return ""
}

func (e *node) setInputTraits(set func(*inputTraits)) *node {
	set(&e.inputOptions)
	if ed := e.st.editor; ed != nil && e.flags&flagEditable != 0 {
		ed.inputOptions = e.inputOptions.nativeOptions()
	}
	return e
}

// Keyboard chooses the on-screen keyboard's layout for a text input, on
// phones and tablets.
func (e *node) Keyboard(k KeyboardType) *node {
	return e.setInputTraits(func(o *inputTraits) { o.Keyboard = name(keyboardNames[:], k) })
}

// ReturnKey sets what the return key of an on-screen keyboard says, for a
// single-line input. It submits the input as Enter does; text areas keep
// their newlines.
func (e *node) ReturnKey(k ReturnKeyType) *node {
	return e.setInputTraits(func(o *inputTraits) { o.Return = name(returnNames[:], k) })
}

// Capitalize sets what an on-screen keyboard capitalizes as one types.
func (e *node) Capitalize(c Capitalization) *node {
	return e.setInputTraits(func(o *inputTraits) { o.Capitalization = name(capitalizeNames[:], c) })
}

// Autocorrect turns an on-screen keyboard's corrections and suggestions on
// or off for a text input, on by default where the keyboard has them.
func (e *node) Autocorrect(on bool) *node {
	return e.setInputTraits(func(o *inputTraits) {
		o.Correction = "off"
		if on {
			o.Correction = "on"
		}
	})
}

// Autofill tells the system what a text input holds, for it to offer it
// from its passwords and contacts, as with iOS's Passwords above the
// keyboard.
func (e *node) Autofill(k AutofillKind) *node {
	return e.setInputTraits(func(o *inputTraits) { o.Content = name(autofillNames[:], k) })
}

// KeyboardDismiss chooses how scrolling dismisses the on-screen keyboard
// while the input has the focus. It is the fork's; #940 has no
// counterpart.
func (e *node) KeyboardDismiss(m KeyboardDismissMode) *node {
	return e.setInputTraits(func(o *inputTraits) { o.Dismiss = string(m) })
}
