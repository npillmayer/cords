## Formatting Styled Text on Monospaced Output Devices

Package formatter formats styled text on output devices with
fixed-width fonts. It is intended for situations where the application is
responsible for the visual representation (as opposed to output to a
browser, which usually addresses the complications of text by itself,
transparently for applications).
Think of this package in terms of `fmt.Println` for styled, bi-directional
text.