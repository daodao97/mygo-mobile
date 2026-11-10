# keyboardbar

A row of keyboard actions for a phone's software keyboard, as a terminal's
Tab, Esc and Ctrl keys, drawn with native UI's buttons (`ui`): they keep the
focus of the input in use and the composition in it. `ModifierLatch` keeps
software modifiers for `ui.Element.InputModifiers`.

```go
if field.Focused() {
    keyboardbar.Bar(c, latch.Decorate(actions, modifierOf), &expanded, onAction)
}
```

Draw it at the bottom of the page: phones lay the content out above the
keyboard while it shows. Go only; it works on every platform, desktops and
headless tests included.
