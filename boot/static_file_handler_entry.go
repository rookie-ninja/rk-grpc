// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-common/error"
	"github.com/rookie-ninja/rk-entry/entry"
	"go.uber.org/zap"
	"html/template"
	"io/fs"
	"math"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// StaticFileHandlerEntryType type of entry
	StaticFileHandlerEntryType = "GrpcStaticFileHandlerEntry"
	// StaticFileHandlerEntryNameDefault name of entry
	StaticFileHandlerEntryNameDefault = "GrpcStaticFileHandlerDefault"
	// StaticFileHandlerEntryDescription description of entry
	StaticFileHandlerEntryDescription = "Internal RK entry which implements static file handler Grpc framework."
)

var exToIcon = map[string]string{
	// folder
	"folder": "folder.png",
	// compressed file
	"ar":  "pkg.png",
	"zip": "pkg.png",
	"rar": "pkg.png",
	"gz":  "pkg.png",
	"xz":  "pkg.png",
	"gz2": "pkg.png",
	"tar": "pkg.png",
	"dep": "pkg.png",
	"rpm": "pkg.png",
	// image file
	"jpg":  "image.png",
	"jpeg": "image.png",
	"png":  "image.png",
	"gif":  "image.png",
	"svg":  "image.png",
	// audio
	"mp3":  "audio.png",
	"wav":  "image.png",
	"ogg":  "image.png",
	"flac": "image.png",
	// pdf
	"pdf": "pdf.png",
	// docs
	"xls":  "doc.png",
	"odt":  "doc.png",
	"ods":  "doc.png",
	"doc":  "doc.png",
	"docx": "doc.png",
	"xlsx": "doc.png",
	"ppt":  "doc.png",
	"txt":  "doc.png",
	// unknown
	"unknown": "file.png",
}

// BootConfigStaticHandler bootstrap config of StaticHandler.
type BootConfigStaticHandler struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Path       string `yaml:"path" json:"path"`
	SourceType string `yaml:"sourceType" json:"sourceType"`
	SourcePath string `yaml:"sourcePath" json:"sourcePath"`
}

// StaticFileHandlerEntry Static file handler entry supports web UI for downloading static files.
type StaticFileHandlerEntry struct {
	EntryName        string                    `yaml:"entryName" json:"entryName"`
	EntryType        string                    `yaml:"entryType" json:"entryType"`
	EntryDescription string                    `yaml:"-" json:"-"`
	Path             string                    `yaml:"path" json:"path"`
	EventLoggerEntry *rkentry.EventLoggerEntry `json:"-" yaml:"-"`
	ZapLoggerEntry   *rkentry.ZapLoggerEntry   `json:"-" yaml:"-"`
	Fs               http.FileSystem           `yaml:"-" json:"-"`
	Template         *template.Template        `json:"-" yaml:"-"`
}

// StaticFileHandlerEntryOption StaticFileHandlerEntry option.
type StaticFileHandlerEntryOption func(*StaticFileHandlerEntry)

// WithEventLoggerEntryCommonService Provide path.
func WithPathStatic(path string) StaticFileHandlerEntryOption {
	return func(entry *StaticFileHandlerEntry) {
		if len(path) > 0 {
			entry.Path = path
		}
	}
}

// WithEventLoggerEntryCommonService Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntryStatic(eventLoggerEntry *rkentry.EventLoggerEntry) StaticFileHandlerEntryOption {
	return func(entry *StaticFileHandlerEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// WithZapLoggerEntryCommonService Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntryStatic(zapLoggerEntry *rkentry.ZapLoggerEntry) StaticFileHandlerEntryOption {
	return func(entry *StaticFileHandlerEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// WithNameStatic Provide name.
func WithNameStatic(name string) StaticFileHandlerEntryOption {
	return func(entry *StaticFileHandlerEntry) {
		if len(name) > 0 {
			entry.EntryName = name
		}
	}
}

// WithFileSystemStatic Provide file system implementation.
func WithFileSystemStatic(fs http.FileSystem) StaticFileHandlerEntryOption {
	return func(entry *StaticFileHandlerEntry) {
		entry.Fs = fs
	}
}

// NewStaticFileHandlerEntry Create new static file handler entry with options.
func NewStaticFileHandlerEntry(opts ...StaticFileHandlerEntryOption) *StaticFileHandlerEntry {
	entry := &StaticFileHandlerEntry{
		EntryName:        StaticFileHandlerEntryNameDefault,
		EntryType:        StaticFileHandlerEntryType,
		EntryDescription: StaticFileHandlerEntryDescription,
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		Template:         template.New("rk-static"),
		Fs:               http.Dir(""),
		Path:             "/rk/v1/static",
	}

	for i := range opts {
		opts[i](entry)
	}

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	// Deal with Path
	// add "/" at start and end side if missing
	if !strings.HasPrefix(entry.Path, "/") {
		entry.Path = "/" + entry.Path
	}

	if !strings.HasSuffix(entry.Path, "/") {
		entry.Path = entry.Path + "/"
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = CommonServiceEntryNameDefault
	}

	return entry
}

// Bootstrap entry.
func (entry *StaticFileHandlerEntry) Bootstrap(ctx context.Context) {
	// parse template
	if _, err := entry.Template.Parse(string(readFileFromPkger("/assets/static/index.tmpl"))); err != nil {
		rkcommon.ShutdownWithError(err)
	}
}

// Interrupt entry.
func (entry *StaticFileHandlerEntry) Interrupt(ctx context.Context) {
	// Noop
}

// GetName Get name of entry.
func (entry *StaticFileHandlerEntry) GetName() string {
	return entry.EntryName
}

// GetType Get entry type.
func (entry *StaticFileHandlerEntry) GetType() string {
	return entry.EntryType
}

// GetDescription Get description of entry.
func (entry *StaticFileHandlerEntry) GetDescription() string {
	return entry.EntryDescription
}

// String Stringfy entry.
func (entry *StaticFileHandlerEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// MarshalJSON Marshal entry.
func (entry *StaticFileHandlerEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":        entry.EntryName,
		"entryType":        entry.EntryType,
		"entryDescription": entry.EntryDescription,
		"path":             entry.Path,
		"zapLoggerEntry":   entry.ZapLoggerEntry.GetName(),
		"eventLoggerEntry": entry.EventLoggerEntry.GetName(),
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Not supported.
func (entry *StaticFileHandlerEntry) UnmarshalJSON([]byte) error {
	return nil
}

// GetFileHandler handles requests sent from user.
func (entry *StaticFileHandlerEntry) GetFileHandler(w http.ResponseWriter, r *http.Request) {
	// Trim prefix with path user defined in order to get file path
	p := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, entry.Path), "/")
	if len(p) < 1 {
		p = "/"
	}
	p = path.Join("/", p)

	var file http.File
	var err error

	// open file
	if file, err = entry.Fs.Open(p); err != nil {
		entry.ZapLoggerEntry.GetLogger().Warn("failed to open file", zap.Error(err))

		res := rkerror.New(
			rkerror.WithHttpCode(http.StatusInternalServerError),
			rkerror.WithMessage("failed to open file"),
			rkerror.WithDetails(err))

		http.Error(w, res.Err.Error(), http.StatusInternalServerError)
		return
	}

	// get file info
	fileInfo, err := file.Stat()
	if err != nil {
		entry.ZapLoggerEntry.GetLogger().Warn("failed to stat file", zap.Error(err))

		res := rkerror.New(
			rkerror.WithHttpCode(http.StatusInternalServerError),
			rkerror.WithMessage("failed to stat file"),
			rkerror.WithDetails(err))

		http.Error(w, res.Err.Error(), http.StatusInternalServerError)
		return
	}

	// list files if file is directory
	if fileInfo.IsDir() {
		infos, _ := file.Readdir(math.MaxInt32)
		files := make([]*fileResp, 0)

		for _, v := range infos {
			files = append(files, &fileResp{
				isDir:    v.IsDir(),
				Icon:     base64.StdEncoding.EncodeToString(readFileFromPkger(path.Join("/assets/static/icons", getIconPath(v)))),
				FileUrl:  path.Join(entry.Path, p, v.Name()),
				FileName: v.Name(),
				Size:     v.Size(),
				ModTime:  v.ModTime(),
			})
		}

		sortFiles(files)
		resp := &resp{
			PrevPath: path.Join(entry.Path, path.Dir(p)),
			PrevIcon: base64.StdEncoding.EncodeToString(readFileFromPkger(path.Join("/assets/static/icons/folder.png"))),
			Path:     p,
			Files:    files,
		}

		buf := new(bytes.Buffer)
		if err := entry.Template.ExecuteTemplate(buf, "index", resp); err != nil {
			entry.ZapLoggerEntry.GetLogger().Warn("failed to execute template", zap.Error(err))

			res := rkerror.New(
				rkerror.WithHttpCode(http.StatusInternalServerError),
				rkerror.WithMessage("failed to execute template"),
				rkerror.WithDetails(err))

			http.Error(w, res.Err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	} else {
		// make browser download file
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name()))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeContent(w, r, path.Base(p), time.Now(), file)
	}
}

// sort file response
func sortFiles(res []*fileResp) {
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].isDir && res[j].isDir {
			return strings.Compare(res[i].FileName, res[j].FileName) < 0
		}

		if res[i].isDir {
			return true
		}

		if res[j].isDir {
			return false
		}

		return strings.Compare(res[i].FileName, res[j].FileName) < 0
	})
}

// get icon path based on file information
func getIconPath(info fs.FileInfo) string {
	if info.IsDir() {
		return exToIcon["folder"]
	}

	ex := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
	res := exToIcon[ex]

	if len(res) < 1 {
		return exToIcon["unknown"]
	}

	return res
}

// response for inner
type resp struct {
	PrevPath string
	PrevIcon string
	Path     string
	Files    []*fileResp
}

// file response for inner
type fileResp struct {
	isDir    bool
	FileName string
	FileUrl  string
	Icon     string
	Size     int64
	ModTime  time.Time
}
