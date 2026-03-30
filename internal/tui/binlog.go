package tui

import (
	"encoding/binary"
	"io"
	"os"
)

type BinLog struct {
	FileName string
	fp       *os.File
}

func (blog *BinLog) Open() (err error) {
	blog.fp, err = os.OpenFile(blog.FileName, os.O_RDWR|os.O_CREATE, 0o644)
	return err
}

func (blog *BinLog) Close() error {
	return blog.fp.Close()
}

func (blog *BinLog) Write(ent entry) error {
	_, err := blog.fp.Write(ent.encode())
	return err
}

func (blog *BinLog) Rewrite(entries []entry) error {
	if err := blog.fp.Truncate(0); err != nil {
		return err
	}
	if _, err := blog.fp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	for _, ent := range entries {
		if _, err := blog.fp.Write(ent.encode()); err != nil {
			return err
		}
	}
	return nil
}

func (blog *BinLog) Read(ent entry) (eof bool, err error) {
	err = ent.decode(blog.fp)
	if err == io.EOF {
		return true, nil
	} else if err != nil {
		return false, err
	} else {
		return false, nil
	}
}

type entry interface {
	encode() []byte
	decode(r io.Reader) error
}

type termEntry struct {
	cmd string
}

func (ent *termEntry) encode() []byte {
	data := make([]byte, 4+len(ent.cmd))
	binary.LittleEndian.PutUint32(data[0:4], uint32(len(ent.cmd)))
	copy(data[4:], ent.cmd)
	return data
}

func (ent *termEntry) decode(r io.Reader) error {
	var header = make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	cmdLen := int(binary.LittleEndian.Uint32(header[0:4]))

	data := make([]byte, cmdLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}

	ent.cmd = string(data)
	return nil
}

type noteEntry struct {
	id    uint64
	Title string
	Body  string
}

func (ent *noteEntry) encode() []byte {
	titleLen := len(ent.Title)
	bodyLen := len(ent.Body)
	data := make([]byte, 16+titleLen+bodyLen)
	binary.LittleEndian.PutUint64(data[0:8], ent.id)
	binary.LittleEndian.PutUint32(data[8:12], uint32(titleLen))
	binary.LittleEndian.PutUint32(data[12:16], uint32(bodyLen))
	copy(data[16:16+titleLen], ent.Title)
	copy(data[16+titleLen:], ent.Body)
	return data
}

func (ent *noteEntry) decode(r io.Reader) error {
	header := make([]byte, 16)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	ent.id = binary.LittleEndian.Uint64(header[0:8])
	titleLen := int(binary.LittleEndian.Uint32(header[8:12]))
	bodyLen := int(binary.LittleEndian.Uint32(header[12:16]))

	title := make([]byte, titleLen)
	body := make([]byte, bodyLen)

	if _, err := io.ReadFull(r, title); err != nil {
		return err
	}
	if _, err := io.ReadFull(r, body); err != nil {
		return err
	}

	ent.Title = string(title)
	ent.Body = string(body)
	return nil
}
