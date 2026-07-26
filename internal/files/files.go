// Package files browses and edits a server's filesystem over SFTP.
//
// SFTP rides the SSH connection that already exists, so this needs no agent,
// no extra port and no new credential — it works on every server the control
// plane can already reach.
package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

var (
	ErrNotSupported = errors.New("this server has no filesystem to browse")
	ErrTooLarge     = errors.New("file is too large to open here")
	ErrNotText      = errors.New("file does not look like text")
	ErrBadPath      = errors.New("path must be absolute")
)

// MaxEditableSize bounds what the editor will load. Anything bigger is a log
// or an archive, and pulling it through the browser helps nobody.
const MaxEditableSize = 2 << 20 // 2 MiB

// Entry is one item in a directory listing.
type Entry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	IsDir      bool      `json:"isDir"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modifiedAt"`
	// IsSymlink is shown because following one can leave the directory you
	// think you are in.
	IsSymlink bool `json:"isSymlink"`
}

// Listing is a directory and its contents.
type Listing struct {
	Path    string  `json:"path"`
	Parent  string  `json:"parent"`
	Entries []Entry `json:"entries"`
}

// Content is a text file opened for editing.
type Content struct {
	Path string `json:"path"`
	Text string `json:"text"`
	Size int64  `json:"size"`
	Mode string `json:"mode"`
}

type Service struct {
	store  *store.Store
	sealer *secret.Sealer
}

func NewService(st *store.Store, sealer *secret.Sealer) *Service {
	return &Service{store: st, sealer: sealer}
}

// session bundles the SSH connection with its SFTP subsystem so both are
// closed together.
type session struct {
	client *sshexec.Client
	sftp   *sftp.Client
}

func (s *session) Close() {
	_ = s.sftp.Close()
	_ = s.client.Close()
}

func (s *Service) connect(ctx context.Context, serverID string) (*session, error) {
	server, err := s.store.GetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server.Kind != store.ConnectionSSH {
		return nil, ErrNotSupported
	}

	password, err := s.sealer.Open(server.SealedPassword)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored password: %w", err)
	}
	privateKey, err := s.sealer.Open(server.SealedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored key: %w", err)
	}

	client, err := sshexec.Dial(ctx, sshexec.Config{
		Host:       server.Host,
		Port:       server.Port,
		User:       server.User,
		Password:   password,
		PrivateKey: privateKey,
	})
	if err != nil {
		return nil, err
	}

	fs, err := client.OpenSFTP()
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	return &session{client: client, sftp: fs}, nil
}

// cleanPath rejects anything that is not an absolute path, so a crafted
// "../.." cannot walk somewhere the caller did not name.
func cleanPath(raw string) (string, error) {
	if raw == "" {
		raw = "/"
	}
	if !strings.HasPrefix(raw, "/") {
		return "", ErrBadPath
	}
	return path.Clean(raw), nil
}

func (s *Service) List(ctx context.Context, serverID, dir string) (*Listing, error) {
	target, err := cleanPath(dir)
	if err != nil {
		return nil, err
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	infos, err := sess.sftp.ReadDir(target)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, Entry{
			Name:       info.Name(),
			Path:       path.Join(target, info.Name()),
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			Mode:       info.Mode().Perm().String(),
			ModifiedAt: info.ModTime().UTC(),
			IsSymlink:  info.Mode()&os.ModeSymlink != 0,
		})
	}

	// Directories first, then alphabetical — how every file browser behaves,
	// and the order people scan in.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return &Listing{Path: target, Parent: path.Dir(target), Entries: entries}, nil
}

func (s *Service) Read(ctx context.Context, serverID, file string) (*Content, error) {
	target, err := cleanPath(file)
	if err != nil {
		return nil, err
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	info, err := sess.sftp.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a folder, not a file", target)
	}
	if info.Size() > MaxEditableSize {
		return nil, ErrTooLarge
	}

	handle, err := sess.sftp.Open(target)
	if err != nil {
		return nil, err
	}
	defer func() { _ = handle.Close() }()

	data, err := io.ReadAll(io.LimitReader(handle, MaxEditableSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxEditableSize {
		return nil, ErrTooLarge
	}
	if isBinary(data) {
		return nil, ErrNotText
	}

	return &Content{
		Path: target,
		Text: string(data),
		Size: info.Size(),
		Mode: info.Mode().Perm().String(),
	}, nil
}

// Write replaces a file's contents, preserving its permissions.
func (s *Service) Write(ctx context.Context, serverID, file, text string) error {
	target, err := cleanPath(file)
	if err != nil {
		return err
	}
	if int64(len(text)) > MaxEditableSize {
		return ErrTooLarge
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return err
	}
	defer sess.Close()

	// Keep the existing mode: writing a config back as 0644 when it was 0600
	// would quietly widen access to it.
	var mode os.FileMode = 0o644
	if info, err := sess.sftp.Stat(target); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a folder, not a file", target)
		}
		mode = info.Mode().Perm()
	}

	handle, err := sess.sftp.Create(target)
	if err != nil {
		return err
	}

	if _, err := handle.Write([]byte(text)); err != nil {
		_ = handle.Close()
		return err
	}
	if err := handle.Close(); err != nil {
		return err
	}

	return sess.sftp.Chmod(target, mode)
}

// Download streams a file to the caller without holding it in memory.
func (s *Service) Download(ctx context.Context, serverID, file string, w io.Writer) (string, error) {
	target, err := cleanPath(file)
	if err != nil {
		return "", err
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return "", err
	}
	defer sess.Close()

	handle, err := sess.sftp.Open(target)
	if err != nil {
		return "", err
	}
	defer func() { _ = handle.Close() }()

	if _, err := io.Copy(w, handle); err != nil {
		return "", err
	}
	return path.Base(target), nil
}

func (s *Service) Remove(ctx context.Context, serverID, target string) error {
	cleaned, err := cleanPath(target)
	if err != nil {
		return err
	}
	if cleaned == "/" {
		return errors.New("refusing to remove the root directory")
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return err
	}
	defer sess.Close()

	info, err := sess.sftp.Stat(cleaned)
	if err != nil {
		return err
	}
	if info.IsDir() {
		// RemoveDirectory fails on a non-empty directory, which is the safe
		// behaviour: recursive deletion is not something to offer casually.
		return sess.sftp.RemoveDirectory(cleaned)
	}
	return sess.sftp.Remove(cleaned)
}

// isBinary guesses whether data is text. A NUL byte in the first block is the
// same heuristic grep and git use, and it is right often enough.
func isBinary(data []byte) bool {
	limit := min(len(data), 8000)
	for i := range limit {
		if data[i] == 0 {
			return true
		}
	}
	return false
}
