Changelog
=========
( English / [Japanese](./CHANGELOG_ja.md) )

- Improve hex view column rendering for better readability across terminals where bold text can appear too prominent: (#82)
  - Change hex view layout to match `hexdump -C` style
  - Remove bold styling from hex view
- Update selection highlighting (#83) :
  - Use white text on blue background by default
  - Fall back to reverse video when `NO_COLOR` is set

v0.10.0
-------
Mar 27, 2026

- Add Ctrl-G to cancel current mode (selection / direct edit) and return to view mode (#77)
- Add Tab / Shift+Tab to move the cursor to the start of the next/previous UTF-8 character (#78)
- Added experimental support for macOS and FreeBSD binaries in the distribution package. (#79)

v0.9.0
-------
Mar 21, 2026

### Bug fixes

- Fix an issue where the space between the hex column and the character column was one character shorter on the last line (#55)
- Fix data near EOF not shown on initial display (appears after pressing a key) (#60)
- Fix an issue where the screen was not fully updated when the number of visible lines decreased. (#66)

### New features

- Add selection and editing features (#56)
  - Selection mode (`v`)
  - Yank (`y`) and delete+yank (`d`)
  - `p` and `P` support multi-byte paste
- Add search functionality (#64)
  - Supports searching for byte sequences and strings
  - `/` and `?` search forward/backward
  - `n` and `N` repeat the previous search
- Respect `NO_COLOR` environment variable to disable colored output (per no-color.org) (#73)

### Improve

- Allow `-` as the output file name to write to standard output
  (refused when standard output is a terminal) (#57)
- Use standard error for screen output when standard output is redirected (#57)
- Make `a` move the cursor to the inserted byte (#62)
- Allow interrupting save operation with Ctrl-C (#71)
- Improve color scheme to work well on both dark and light backgrounds (#72)

v0.8.0
------
Mar 13, 2026

### Bug fixes

- Fixed an issue where the version string was empty when built without GNU Make.  
  The version string is now updated via `make bump` during the release process. (#33)
- Fixed an issue where the progress animation was cleared at the wrong position during long save operations. ([go-inline-animation#6], [go-inline-animation#7], #36)
- Preserve original file permissions when overwriting files (#37,#42)

### New Features

- Add direct edit mode that allows directly overwriting hexadecimal values under the cursor. Toggle with `Shift`+`R` (switches from the traditional command mode). (#43)
- Indicate READONLY files explicitly when prompting for overwrite confirmation (#37)
- Changed file saving to use a temporary file until writing completes, eliminating any window where the original file could be left in a partial state. (#35)
- Add `Shift`+`I` (insert 0x00 to the left of the cursor) and `Shift`+`A` (insert 0x00 to the right of the cursor). (#50)
- After UNDO, move the cursor to the address of the reverted change. (#51)

### Documents

- Rename release note files to CHANGELOG.md and CHANGELOG\_ja.md. (#34)

[go-inline-animation#6]: https://github.com/nyaosorg/go-inline-animation/pull/6
[go-inline-animation#7]: https://github.com/nyaosorg/go-inline-animation/pull/7

v0.7.1
------
Feb 14, 2026

- Improved build portability by replacing local helper tools with 'go run' (make release, make manifest). (#28)
- Move from https://github.com/hymkor/binview to https://github.com/hymkor/bine (#31)

v0.7.0
------
Feb 6, 2026

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
