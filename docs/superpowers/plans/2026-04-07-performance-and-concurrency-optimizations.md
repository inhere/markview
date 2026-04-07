# MarkView Performance and Concurrency Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor and optimize MarkView's core logic to reduce memory allocations, improve search performance, fix concurrent watcher bugs, and establish a robust server structure.

**Architecture:** We will introduce a global singleton for the `goldmark` Markdown parser to avoid repeated initialization. We will also refactor the `fsnotify` watcher to use a single goroutine with channel-based debouncing to eliminate race conditions and reduce goroutine churn. Finally, we will refactor HTTP server initialization to include proper timeouts, and extract common page rendering logic to reduce code duplication.

**Tech Stack:** Go 1.22+, `fsnotify`, `yuin/goldmark`, `net/http`

---

### Task 1: Refactor HTTP Server Initialization with Timeouts

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace DefaultServer with an explicit `http.Server`**
  Modify `main.go` around line 73. Replace `log.Fatal(http.ListenAndServe(":"+config.Cfg.PortStr(), newServerMux()))` with an `http.Server` struct that includes `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.

- [ ] **Step 2: Add context/timeouts to server shutdown (if needed, or just set timeouts)**
  Ensure the server struct looks like:
  ```go
  server := &http.Server{
      Addr:         ":" + config.Cfg.PortStr(),
      Handler:      newServerMux(),
      ReadTimeout:  5 * time.Second,
      WriteTimeout: 10 * time.Second,
      IdleTimeout:  120 * time.Second,
  }
  log.Fatal(server.ListenAndServe())
  ```

- [ ] **Step 3: Run the project to verify it still boots up successfully**
  Run `go run main.go` and verify server starts at the given port without immediate crash.

- [ ] **Step 4: Commit**
  `git commit -m "refactor: configure http server with strict timeouts"`

---

### Task 2: Optimize Markdown Parser (Extract to Singleton)

**Files:**
- Modify: `internal/handlers/page_handler.go`

- [ ] **Step 1: Define a global `goldmark.Markdown` variable and an init function**
  In `internal/handlers/page_handler.go`, create a global `var mdParser goldmark.Markdown` and an `init()` block (or a `sync.Once` wrapper) to initialize it once with the existing options (GFM, Emoji, Meta, HTML unsafe, etc.).

- [ ] **Step 2: Refactor `renderMarkdownContent` to use the global parser**
  Remove the `md := goldmark.New(...)` logic inside `renderMarkdownContent` and replace it with `err = mdParser.Convert(mdData, &buf)`.

- [ ] **Step 3: Run the project and verify markdown still renders**
  Run `go run main.go example` and load `http://localhost:6100/` in browser to confirm markdown is successfully parsed.

- [ ] **Step 4: Commit**
  `git commit -m "perf: reuse goldmark parser instance to reduce allocations"`

---

### Task 3: Fix FSWatcher Concurrency and Debounce Logic

**Files:**
- Modify: `internal/handlers/watch_handler.go`

- [ ] **Step 1: Replace global `debounceTimer` and `debounceMutex` with a channel-based architecture**
  Remove `debounceTimer`, `debounceMutex`, and `pendingFiles`.
  Introduce a `debounce` function that accepts a `chan fsnotify.Event` and processes events with a `time.Timer` in a dedicated goroutine, calling `broadcastJSON(files)` when the timer expires.

- [ ] **Step 2: Refactor `handleFileChange`**
  Instead of doing mutex locks and timer resets directly, `handleFileChange` should just send the event to the debounce channel.
  Update the interval from `2*time.Second` to `200*time.Millisecond` for faster live reloads.

- [ ] **Step 3: Test file watching**
  Run `go run main.go example`. Edit `example/basics.md` and verify the terminal logs the change and the SSE reload message is sent without crashing or race conditions.

- [ ] **Step 4: Commit**
  `git commit -m "fix: refactor watcher debounce using channels for thread safety"`

---

### Task 4: Optimize Search Implementation

**Files:**
- Modify: `internal/handlers/search_handler.go`

- [ ] **Step 1: Optimize `SearchTerms` struct and `parseSearchTerms`**
  Update `SearchTerms` to include `IncludeLower` and `ExcludeLower`.
  In `parseSearchTerms`, pre-calculate `strings.ToLower(word)` for all include/exclude keywords so they aren't repeatedly converted during the search loop.

- [ ] **Step 2: Refactor `lineMatchesMatch`**
  Replace `strings.ToLower(ex)` and `strings.ToLower(inc)` inside the loop with the pre-calculated `ExcludeLower` and `IncludeLower` fields.

- [ ] **Step 3: Refactor `sortFileTreeNodes` in `page_handler.go` (Related minor fix)**
  In `page_handler.go`'s `sortFileTreeNodes`, replace the `strings.ToLower` allocations with `strings.Compare(strings.ToLower(...))` or an allocation-free case-insensitive comparison using `strings.EqualFold` for equality and manual lowercasing if not equal, or just keep it simple but avoid unnecessary variables. (Actually, `strings.Compare` doesn't do case-insensitive natively, a quick optimization is acceptable).

- [ ] **Step 4: Run the search API to verify results**
  Trigger `/api/search?q=test` (or similar endpoint) via curl or the browser to verify search logic still works identically.

- [ ] **Step 5: Commit**
  `git commit -m "perf: optimize search text processing to reduce string allocations"`

---

### Task 5: DRY Page Rendering Logic

**Files:**
- Modify: `internal/handlers/page_handler.go`

- [ ] **Step 1: Extract `buildPageData` helper function**
  Extract the identical logic from `renderMarkdown` and `renderMainContent` (statting the file, formatting dates, resolving relative paths, checking file sizes, calling `renderMarkdownContent`) into a common function `func buildPageData(filePath string) (*PageData, error)`.

- [ ] **Step 2: Refactor `renderMarkdown` and `renderMainContent`**
  Update both functions to call `buildPageData(filePath)`.
  `renderMainContent` will only execute `template-main.html` and write the buffer.
  `renderMarkdown` will also fetch the `fileTree`, execute `template-main.html`, and then pass the result to `template.html`.

- [ ] **Step 3: Test page loads**
  Load `http://localhost:6100/` to test full page render.
  Load `http://localhost:6100/?q=main` to test main content partial render.

- [ ] **Step 4: Commit**
  `git commit -m "refactor: extract common page data building logic (DRY)"`