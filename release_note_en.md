Release notes
=============

- Changed `G` (`Shift`-`G`) to move to the end of the currently loaded data instead of waiting for all data to be read. (#11)
- Prevent key input responsiveness from being blocked even when data reading stalls. (#13)
- Renamed the executable from `binview` to `bine`, and updated the product name to Bine. (#14)
  - (Planned) When the stable version of `bine` is released:
    - Rename the repository from `binview` to `bine`
    - Update `go.mod`, `go.sum`, import paths, README URLs, and the Scoop manifest accordingly
- Echo the `y` input to the screen during overwrite confirmation. (#17)
- Display a text animation while waiting for a save operation to complete. (#17)
- When executing the `q` command, prompt whether to save the changes (#18)
- Changed the `Esc` key from application exit to a prefix-only key to prevent unintended behavior caused by split input sequences. (#21)
- Readline: Treat `Ctrl`+`G` and `Meta`+`Ctrl`+`G` as cancel input instead of `Esc` (#24)
- Remove the default save file name (`output.new`) when reading from standard input. (#26)

v0.6.3
------
Jan 2, 2022

- (#3) Do not display `U+0080`-`U+009F`, the Unicode Characters in the 'Other, Control' Category

v0.6.2
-------
Nov 30, 2021

- Fix: on Linux, `w`: output was zero bytes.

v0.6.1
------
Nov 28, 2021

- Fix: on ANSI encoding, the byte-length of ANK was counted as 2-bytes

v0.6.0
-------
Nov 26, 2021

- `i`/`a`: `"string"` or `U+nnnn`: insert with the current encoding
- Detect the encoding if data starts with U+FEFF
- `u` : implement the undo

v0.5.0
------
Nov 13, 2021

- ALT-L: Change the character encoding to UTF16LE
- ALT-B: Change the character encoding to UTF16BE
- Show some unicode's name(ByteOrderMark,ZeroWidthjoin) on the status line
- i: insert multi bytes data (for example: `0xFF`,`U+0000`,`"utf8string"`)
- a: append multi bytes data (for example: `0xFF`,`U+0000`,`"utf8string"`)
- Support history on getline

v0.4.1
------
Oct 15, 2021

- Fix: `$` does not move the cursor when the current line is less then 16 bytes

v0.4.0
------
Oct 9, 2021

- Update status-line even if no keys are typed
- ALT-A: Change the character encoding to the current codepage (Windows-Only)
- ALT-U: Change the character encoding to UTF8 (default)

v0.3.0
------
Sep 23, 2021

- Fix the problem that the utf8-rune on the line boundary could not be drawn
- `w`: restore the last saved filename as the next saving
- `w`: show `canceled` instead of `^C` when ESCAPE key is pressed
- Display CR, LF, TAB with half width arrows
- Read data while waiting key typed
- Improve the internal data structure and be able to read more huge data
- Fix: the text color remained yellow even after the program ended

v0.2.1
------
Jul 5, 2021

- (#1) Fix the overflow that pointer to seek the top of the rune is decreased less than zero (Thx @spiegel-im-spiegel)
- If the cursor is not on utf8 sequences, print `(not utf8)`
- If the parameter is a directory, show error and quit immediately instead of hanging

v0.2.0
------
Jul 5, 2021

- Status line:
    - current rune's codepoint
    - changed/unchanged mark
- Implement key feature
    - p (paste 1 byte the rightside of the cursor)
    - P (paste 1 byte the leftside of the cursor)
    - a (append '\0' at the rightside of the cursor)
- Update library [go-readline-ny to v0.4.13](https://github.com/zetamatta/go-readline-ny/releases/tag/v0.4.13)

v0.1.1
------
Dec 28, 2020

- Did go mod init to fix the problem not able to build because the incompatibility of go-readline-ny between v0.2.6 and v0.2.8
- The binary executable of v0.1.0 has no problems.

v0.1.0
------
Nov 8, 2020

- The first version
