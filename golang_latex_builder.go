package golang_latex_builder

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func Build(project_name, repo_url, commit, build_dir_name, out_dir_name string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	build_dir := path.Join(pwd, build_dir_name)
	out_dir := path.Join(pwd, out_dir_name)
	repo_build_dir := path.Join(build_dir, project_name)
	repo_out_dir := path.Join(out_dir, project_name)
	clone_dir := path.Join(repo_build_dir, commit)
	lockfile := filepath.Join(repo_build_dir, "."+commit)
	clone_url, err := GetCloneUrl(repo_url)

	fmt.Println("Preparing environment")
	err = Prepare(repo_build_dir, repo_out_dir, clone_dir, lockfile)

	if err != nil {
		return err
	}

	fmt.Println("Cloning repository")
	err = Clone(clone_url, clone_dir, commit)

	if err != nil {
		return err
	}

	fmt.Println("Building latex project")
	err = DoMake(clone_dir)

	if err != nil {
		return err
	}

	fmt.Println("Copying files")
	err = CopyPDF(repo_build_dir, repo_out_dir)

	if err != nil {
		return err
	}

	fmt.Println("Cleanup")
	err = Cleanup(lockfile)

	if err != nil {
		return err
	}
	return nil
}

func GetCloneUrl(repo_url string) (string, error) {
	if strings.HasPrefix(repo_url, "https://") || strings.HasPrefix(repo_url, "git") {
		if !strings.HasSuffix(repo_url, ".git") {
			return repo_url + ".git", nil
		} else {
			return repo_url, nil
		}
	} else {
		return "", errors.New("Invalid repository")
	}
}

func Makedir(path string) {
	err := os.MkdirAll(path, 0700)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error when creating dir: %s -> ", path), err)
	}
}

func RemoveContents(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, n := range names {
		err = os.RemoveAll(filepath.Join(path, n))
		if err != nil {
			return err
		}
	}
	return nil
}

func Prepare(repo_build_dir, repo_out_dir, clone_dir, lockfile string) error {

	Makedir(repo_build_dir)
	Makedir(repo_out_dir)

	f, err := os.OpenFile(lockfile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if pe, ok := err.(*os.PathError); ok && pe.Err == os.ErrExist {
		return err
	} else {
		f.Close()
	}
	return nil
}

func Clone(clone_url, clone_dir, commit string) error {
	if _, err := os.Stat(clone_dir); err == nil {
		err_rmdir := os.RemoveAll(clone_dir)
		if err_rmdir != nil {
			return err_rmdir
		}
	}

	cmd := exec.Command("git", "clone", clone_url, clone_dir)
	err := cmd.Run()
	if err != nil {
		return err
	}
	original_pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(clone_dir)

	if err != nil {
		return err
	}
	cmd = exec.Command("git", "submodule", "init")
	err = cmd.Run()

	if err != nil {
		return err
	}

	cmd = exec.Command("git", "submodule", "update")
	err = cmd.Run()

	if err != nil {
		return err
	}

	cmd = exec.Command("git", "checkout", commit)
	err = cmd.Run()

	if err != nil {
		return err
	}
	err = os.Chdir(original_pwd)
	if err != nil {
		return err
	}

	return nil
}

func DoMake(workdir string) error {
	cmd := exec.Command("make")
	original_pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(workdir)

	if err != nil {
		return err
	}

	err = cmd.Run()
	if err != nil {
		return err
	}
	err = os.Chdir(original_pwd)
	if err != nil {
		return err
	}

	return nil
}

func CopyPDF(workdir string, pdfdir string) error {
	RemoveContents(pdfdir)
	d, err := os.Open(workdir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	var cerr error
	for _, name := range names {
		if strings.HasSuffix(name, ".pdf") {
			in, err := os.Open(filepath.Join(workdir, name))
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.Create(filepath.Join(pdfdir, name))
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, in)
			cerr = out.Close()
			if err != nil {
				return err
			}
		}
	}
	return cerr
}

func Cleanup(lockfile string) error {
	err := os.Remove(lockfile)
	if err != nil {
		return err
	}
	return nil
	//i have to implement something else here, like a make clean execution
}
