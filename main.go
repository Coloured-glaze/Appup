package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/Coloured-glaze/Appup/file"
)

var Version = "1.0.0-beta"

type data struct {
	Name string `json:"name"`
}

func main() {
	version, up, err := getver(Version) // version
	if err != nil {
		log.Println("Getver Error: ", err)
		return
	}
	upath, _ := os.Getwd()
	if up {
		OS := runtime.GOOS
		arch := runtime.GOARCH
		log.Printf("版本为 %v for %v %v\n正在更新到: %v for %v %v ...\n",
			version, OS, arch, Version, OS, arch)

		if OS == "windows" {

			upath += "\\data\\Update\\"
			path := "App_" + OS + "_" + arch + ".zip"
			path2 := upath + path

			err = download(path, path2, upath, version)
			if err != nil {
				log.Println("Download Error: ", err)
				return
			}
			err = unzip(path2, upath)
			if err != nil {
				log.Println("unzip Error: ", err)
				return
			}
			err = forkwin(upath)
			if err != nil {
				log.Println("fork Error: ", err)
				return
			}
			log.Println("更新完成!")
			os.Exit(0)
		} else {
			uppath := upath + "/data/Update/"
			path := "App_" + OS + "_" + arch + ".tar.gz"
			path2 := uppath + path

			err = download(path, path2, uppath, version)
			if err != nil {
				log.Println("Download Error: ", err)
				return
			}

			err = Decompress(path2, uppath)
			if err != nil {
				log.Println("Decompress Error: ", err)
				return
			}
			err = fork()
			if err != nil {
				log.Println("fork Error: ", err)
				return
			}
			log.Println("更新完成!")
			os.Exit(0)
		}
	} else {
		log.Println("没有版本更新!")
	}

}

// 检查版本更新
func getver(version string) (string, bool, error) {
	resp, err := http.Get(
		"https://api.github.com/repos/Coloured-glaze/Appup/tags")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}
	datas := make([]data, 0, 32)
	err = json.Unmarshal(b, &datas)
	if err != nil {
		return "", false, err
	}
	if len(datas) == 0 {
		return "", false, nil
	}
	if datas[0].Name != version { // 当前不是最新版
		return datas[0].Name, true, nil
	}
	return "", false, err
}

// 下载新版本
func download(path, path2, dpath, version string) error {
	var err error
	if !file.IsExist(dpath) { // 如果文件夹不存在就创建
		err = os.Mkdir(dpath, 0777)
		if err != nil {
			return err
		}
	}
	// 下载的文件重命名为 path2
	err = file.DownloadTo(
		"https://hub.fastgit.xyz/Coloured-glaze/Appup/releases/download/"+version+"/"+path,
		path2, false)
	if err != nil {
		err = file.DownloadTo(
			"https://github.com/Coloured-glaze/Appup/releases/download/"+version+"/"+path,
			path2, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// 解压tar.gz
func Decompress(defile, uppath string) error {
	f, err := os.Open(defile)
	if err != nil {
		return err
	}
	defer f.Close()
	g, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer g.Close()
	t := tar.NewReader(g)

	for {
		next, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		of, err := os.OpenFile(next.Name, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		defer of.Close()

		_, err = io.Copy(of, t)
		if err != nil {
			return err
		}
	}
	n, err := os.Executable()
	if err != nil {
		return err
	}
	err = os.Rename(uppath+"zbp", n) // 重命名并覆盖
	if err != nil {
		return err
	}
	return nil
}

// 解压缩zip
func unzip(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()
	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}
			inFile, err := f.Open() //读取压缩文件
			if err != nil {
				return err
			}
			defer inFile.Close()
			outFile, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()) //创建的新文件
			if err != nil {
				return err
			}
			defer outFile.Close()
			io.Copy(outFile, inFile)
		}
	}
	return err
}

func fork() error {
	path, err := os.Executable() // 可执行文件的绝对路径
	if err != nil {
		return err
	}
	args := []string{path}
	args = append(args, os.Args[1:]...) // 加入命令行参数

	cmd := &exec.Cmd{
		Path:        path,               // 文件的绝对路径
		Args:        args,               // 执行的命令
		Dir:         filepath.Dir(path), // 去除文件名，保留路径
		Env:         os.Environ(),       // 环境变量
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{},
	}
	err = cmd.Start() // 不阻塞
	if err != nil {
		return err
	}
	return nil
}

func forkwin(exename string) error {
	path, err := os.Executable() // 可执行文件的绝对路径
	if err != nil {
		return err
	}
	base := filepath.Base(path)              // 去除路径，保留文件名
	cmdpath, err := exec.LookPath("cmd.exe") // 返回文件的绝对路径
	if err != nil {
		return err
	}

	args := []string{cmdpath, "/c",
		"TIMEOUT /T 3 & copy /Y " + exename + base + " " + path + " & " + base,
	}
	for _, v := range os.Args[1:] { // 加入命令行参数
		args[2] += " " + v
	}

	cmd := &exec.Cmd{
		Path:        args[0],            // 文件的绝对路径
		Args:        args,               // 执行的命令
		Dir:         filepath.Dir(path), // 去除文件名，保留路径
		Env:         os.Environ(),       // 环境变量
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{},
	}
	err = cmd.Start() // 不阻塞地执行
	if err != nil {
		return err
	}
	return nil
}
