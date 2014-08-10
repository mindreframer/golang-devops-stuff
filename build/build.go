/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"time"
	"os"
	"os/exec"
	"path/filepath"
)

var OS = ""          //windows,linux
var ARCH = ""        //32,64
var VS_PLATFORM = "" //set automatically 
var GOROOT = ""      // must match ARCH
var PATH_SEPARATOR = ""
var PATH_SLASH = ""
var VERSION = "1.2.0"

func main() {
	start()
	//
	buildServer()
	buildService()
	copyRootFiles()
	copyDocFiles()
	copyGo()
	buildJava()	
	buildDotnet()
	buildPython()
	createArchive()
	//
	done()
}

// server

func buildServer() {
	emptyln()
	print("Building pubsubsql server...")
	bin := "build/pubsubsql/bin/"
	cd("..")
	rm(serverFileName())
	execute("go", "build")
	cp(serverFileName(), bin+serverFileName())
	cd("build")
	success()
}

func serverFileName() string {
	switch OS {
	case "windows":
		return "pubsubsql.exe"
	case "linux":
		return "pubsubsql"
	}
	return "invalid server name"
}

// service installer

func buildService() {
	emptyln()
	print("Building service/installer...")
	cd("../service/" + OS)
	switch OS {
	case "linux":
		buildServiceLinux()
	default:
		buildServiceWindows()
	}
	cd("../../build")
	success()
}

func buildServiceWindows() {
	bin := "../../build/pubsubsql/bin/"
	execute("msbuild.exe", "/t:Clean,Build", "/p:Configuration=Release", "/p:Platform="+VS_PLATFORM)
	svc := "pubsubsqlsvc.exe"
	cp(svc, bin+svc)
}

func buildServiceLinux() {
	m := "m64"
	if ARCH == "32" {
		m = "m32"
	}
	bin := "../../build/pubsubsql/bin/"
	execute("make", "ARCH="+m)
	svc := "pubsubsqlsvc"
	cp(svc, bin+svc)
}

// copy README LICENSE etc..

func copyRootFiles() {
	emptyln()
	print("Copying root files...")
	var f fileCopy

	switch OS {
	case "windows":
		f.from = "./windows"
		f.to = "./pubsubsql"
		f.cp("README.txt")
		f.cp("CHANGES.txt")
		f.cp("NOTICES.txt")
		f.cp("LICENSE.txt")
	case "linux":
		f.from = "./linux"
		f.to = "./pubsubsql"
		f.cp("README")
		f.cp("CHANGES")
		f.cp("NOTICES")
		f.cp("LICENSE")
	}

	success()
}

func copyDocFiles() {
	emptyln()
	print("Copying doc files...")
	var f fileCopy

	mkdir("./pubsubsql/docs")			
	
	switch OS {
	case "windows":
		f.from = "./windows/docs"
		f.to = "./pubsubsql/docs"
		f.cp("dot.Net_API.html")
		f.cp("Getting_Started.html")
		f.cp("Go_API.html")
		f.cp("Java_API.html")
		f.cp("Python_API.html")
	case "linux":
		f.from = "./linux/docs"
		f.to = "./pubsubsql/docs"
		f.cp("Getting_Started.html")
		f.cp("Go_API.html")
		f.cp("Java_API.html")
		f.cp("Python_API.html")
	}
}

func copyGo() {
	emptyln()
	print("Copying go files...")
	mkdir("./pubsubsql/samples/go/bin")			
	mkdir("./pubsubsql/samples/go/src/github.com/pubsubsql/client")			
	mkdir("./pubsubsql/samples/go/src/github.com/pubsubsql/QuickStart")			
	mkdir("./pubsubsql/samples/go/pkg")			

	var f fileCopy

	// client
	f.from = "../../client"
	f.to = "./pubsubsql/samples/go/src/github.com/pubsubsql/client"
	f.cp("client.go")
	f.cp("client_test.go")
	f.cp("netheader.go")
	f.cp("netheader_test.go")
	f.cp("nethelper.go")
	// QuickStart
	f.from = "../../samples/QuickStart"
	f.to = "./pubsubsql/samples/go/src/github.com/pubsubsql/QuickStart"
	f.cp("QuickStart.go")

	success()
}

func buildJava() {
	emptyln()
	print("Building Java binaries...")

	print("Building All...")
	cd("../../java")
	shell(shellScript("build"))

	cd("../pubsubsql/build")
	// create directories
	mkdir("./pubsubsql/samples/java/bin")			
	mkdir("./pubsubsql/samples/java/lib")			
	//
	mkdir("./pubsubsql/samples/java/Client")		
	mkdir("./pubsubsql/samples/java/Client/src")		
	mkdir("./pubsubsql/samples/java/Client/src/main")		
	mkdir("./pubsubsql/samples/java/Client/src/main/java")		
	mkdir("./pubsubsql/samples/java/Client/src/main/java/pubsubsql")		
	//
	mkdir("./pubsubsql/samples/java/ClientTest")
	mkdir("./pubsubsql/samples/java/ClientTest/src")
	mkdir("./pubsubsql/samples/java/ClientTest/src/main")
	mkdir("./pubsubsql/samples/java/ClientTest/src/main/java")
	mkdir("./pubsubsql/samples/java/ClientTest/src/main/java/pubsubsql")
	//
	mkdir("./pubsubsql/samples/java/QuickStart")
	mkdir("./pubsubsql/samples/java/QuickStart/src")
	mkdir("./pubsubsql/samples/java/QuickStart/src/main")
	mkdir("./pubsubsql/samples/java/QuickStart/src/main/java")
	mkdir("./pubsubsql/samples/java/QuickStart/src/main/java/pubsubsql")
	//
	mkdir("./pubsubsql/samples/java/PubSubSqlGui")
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src")
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src/main")
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src/main/java")
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src/main/java/pubsubsql")
	//
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src/main/resources")
	mkdir("./pubsubsql/samples/java/PubSubSqlGui/src/main/resources/images")

	var f fileCopy

	// copy <.>
	f.from = "../../java"
	f.to = "./pubsubsql/samples/java"
	f.cp("build.bat")
	f.cp("build.sh")
	f.cp("pom.xml")
	f.cp("run-ClientTest.bat")
	f.cp("run-ClientTest.sh")
	f.cp("run-PubSubSqlGui.bat")
	f.cp("run-PubSubSqlGui.sh")
	f.cp("run-QuickStart.bat")
	f.cp("run-QuickStart.sh")

	// copy Client
	f.from = "../../java/Client"
	f.to = "./pubsubsql/samples/java/Client"
	f.cp("pom.xml")
	//
	f.from = "../../java/Client/src/main/java/pubsubsql"
	f.to = "./pubsubsql/samples/java/Client/src/main/java/pubsubsql"
	f.cp("Client.java")
	f.cp("NetHeader.java")
	f.cp("NetHelper.java")
	f.cp("ResponseData.java")

	// copy ClientTest
	f.from = "../../java/ClientTest"
	f.to = "./pubsubsql/samples/java/ClientTest"
	f.cp("pom.xml")
	//
	f.from = "../../java/ClientTest/src/main/java/pubsubsql"
	f.to = "./pubsubsql/samples/java/ClientTest/src/main/java/pubsubsql"
	f.cp("ClientTest.java")

	// copy QuickStart
	f.from = "../../java/QuickStart"
	f.to = "./pubsubsql/samples/java/QuickStart"
	f.cp("pom.xml")
	//
	f.from = "../../java/QuickStart/src/main/java/pubsubsql"
	f.to = "./pubsubsql/samples/java/QuickStart/src/main/java/pubsubsql"
	f.cp("QuickStart.java")

	// copy PubSubSqlGui 
	f.from = "../../java/PubSubSqlGui"
	f.to = "./pubsubsql/samples/java/PubSubSqlGui"
	f.cp("pom.xml")
	//
	f.from = "../../java/PubSubSqlGui/src/main/java/pubsubsql"
	f.to = "./pubsubsql/samples/java/PubSubSqlGui/src/main/java/pubsubsql"
	f.cp("AboutForm.java")
	f.cp("AboutPanel.java")
	f.cp("ConnectForm.java")
	f.cp("ConnectPanel.java")
	f.cp("MainForm.java")
	f.cp("PubSubSQLGUI.java")
	f.cp("Simulator.java")
	f.cp("SimulatorForm.java")
	f.cp("SimulatorPanel.java")
	f.cp("TableDataset.java")
	f.cp("TableView.java")
	//
	// copy PubSubSqlGui/images
	f.from = "../../java/PubSubSqlGui/src/main/resources/images"
	f.to = "./pubsubsql/samples/java/PubSubSqlGui/src/main/resources/images"
	f.cp("Connect.png")
	f.cp("ConnectLocal.png")
	f.cp("Disconnect.png")
	f.cp("Execute2.png")
	f.cp("New.png")
	f.cp("Stop.png")
	
	// copy bin
	f.from = "../../java/bin"  
	f.to = "./pubsubsql/samples/java/bin"
	f.cp("gitempty")
	
	// copy lib
	f.from = "../../java/lib"  
	f.to = "./pubsubsql/samples/java/lib"
	f.cp("gson-2.2.4.jar")
	f.cp("pubsubsql.jar")
	f.cp("pubsubsql-javadoc.jar")
	//
	f.from = "../../java/lib"
	f.to = "./pubsubsql/lib"
	f.cp("gson-2.2.4.jar")
	f.cp("pubsubsql.jar")
	f.cp("pubsubsql-javadoc.jar")

	if OS != "windows" {
		cp("../../java/bin/pubsubsqlgui.jar", "./pubsubsql/bin/pubsubsqlgui.jar")
	}

	success()
}

func buildDotnet() {
	if OS != "windows" {
		return
	}
	emptyln()
	print("Building .Net binaries...")
	//
	cd("../../dotnet")
	execute("msbuild.exe", "All.sln", "/t:Clean,Build", "/p:Configuration=Release", "/p:Platform=Any CPU")
	// create directories
	
	cd("../../pubsubsql/pubsubsql/build/pubsubsql/samples")
	mkdir("dotNet/bin")			
	mkdir("dotNet/Client")			
	mkdir("dotNet/ClientTest")			
	mkdir("dotNet/QuickStart")			
	mkdir("dotNet/PubSubSQLGUI")			
	mkdir("dotNet/PubSubSQLGUI/Properties")			
	mkdir("dotNet/PubSubSQLGUI/images")			

	//root
	cd("../../../..")

	var f fileCopy
	// copy binaries
	f.from = "dotnet/bin"
	f.to = "pubsubsql/build/pubsubsql/bin/"
	f.cp("pubsubsql.dll")
	f.cp("pubsubsqlgui.exe")

	cp("dotnet/All.sln", "pubsubsql/build/pubsubsql/samples/dotNet/All.sln")
	// copy Client
	f.from = "dotnet/Client"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/Client"
	f.cp("Client.csproj")
	f.cp("AssemblyInfo.cs")
	f.cp("Client.cs")
	f.cp("NetHeader.cs")
	f.cp("NetHelper.cs")

	// ClientTest
	f.from = "dotnet/ClientTest"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/ClientTest"
	f.cp("ClientTest.csproj")
	f.cp("AssemblyInfo.cs")
	f.cp("ClientTest.cs")
	f.cp("NetHeaderTest.cs")
	f.cp("NetHelperTest.cs")

	// QuickStart
	f.from = "dotnet/QuickStart"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/QuickStart"
	f.cp("QuickStart.csproj")
	f.cp("AssemblyInfo.cs")
	f.cp("Program.cs")

	// PubSubSQLGUI	
	f.from = "dotnet/PubSubSQLGUI"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/PubSubSQLGUI"
	f.cp("PubSubSQLGUI.csproj")
	f.cp("AboutForm.cs")		
	f.cp("AboutForm.Designer.cs")		
	f.cp("AboutForm.resx")		
	f.cp("ConnectForm.cs")		
	f.cp("ConnectForm.Designer.cs")		
	f.cp("ConnectForm.resx")		
	f.cp("SimulatorForm.cs")		
	f.cp("SimulatorForm.Designer.cs")		
	f.cp("SimulatorForm.resx")		
	f.cp("MainForm.cs")		
	f.cp("MainForm.Designer.cs")		
	f.cp("MainForm.resx")		
	f.cp("Simulator.cs")		
	f.cp("ListViewDataset.cs")		
	f.cp("ListViewDoubleBuffered.cs")		
	f.cp("Program.cs")		
	f.cp("Resources.Designer.cs")		
	// PubSubSQLGUI/Properties
	f.from = "dotnet/PubSubSQLGUI/Properties"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/PubSubSQLGUI/Properties"
	f.cp("AssemblyInfo.cs")		
	f.cp("Resources.Designer.cs")		
	f.cp("Resources.resx")		
	f.cp("Settings.Designer.cs")		
	f.cp("Settings.settings")		
	f.cp("Settings1.Designer.cs")		
	// PubSubSQLGUI/images
	f.from = "dotnet/PubSubSQLGUI/images"
	f.to = "pubsubsql/build/pubsubsql/samples/dotNet/PubSubSQLGUI/images"
	f.cp("Connect.bmp")		
	f.cp("ConnectLocal.bmp")		
	f.cp("Disconnect.bmp")		
	f.cp("Execute.bmp")		
	f.cp("Execute2.bmp")		
	f.cp("New.bmp")		
	f.cp("Stop.bmp")		
	f.cp("pubsub.ico")		

	cd("pubsubsql/build")
	success()
}

func buildPython() {
	emptyln()
	print("Building Python...")
	
	// create directories
	mkdir("./pubsubsql")
	mkdir("./pubsubsql/samples")
	mkdir("./pubsubsql/samples/python")
	mkdir("./pubsubsql/samples/python/src")
	mkdir("./pubsubsql/samples/python/src/pubsubsql")
	mkdir("./pubsubsql/samples/python/src/pubsubsql/net")
	
	var f fileCopy
	
	// copy files: src/*
	f.from = "../../python/src"
	f.to = "./pubsubsql/samples/python/src"
	f.cp("quickstart.py")

	// copy files: src/pubsubsql/*
	f.from = "../../python/src/pubsubsql"
	f.to = "./pubsubsql/samples/python/src/pubsubsql"
	f.cp("__init__.py")
	f.cp("client.py")
	f.cp("testclient.py")

	// copy files: src/pubsubsql/net/*
	f.from = "../../python/src/pubsubsql/net"
	f.to = "./pubsubsql/samples/python/src/pubsubsql/net"
	f.cp("__init__.py")
	f.cp("header.py")
	f.cp("helper.py")
	f.cp("response.py")
	f.cp("testheader.py")
	
	success()
}

// create archive

func createArchive() {
	emptyln()
	print("Archiving files...")
	switch OS {
	case "linux":
		targz(getarchname()+".tar.gz", "./pubsubsql")
	case "windows":
		dozip(getarchname()+".zip", "pubsubsql")
	}
	success()
}

// helpers

func print(str string, v ...interface{}) {
	fmt.Printf(str, v...)
	fmt.Println("")
}

func fail(str string, v ...interface{}) {
	print("ERROR: "+str, v...)
	os.Exit(1)
}

func emptyln() {
	fmt.Println("")
}

func success() {
	print("SUCCESS")
}

func start() {
	// read flags
	flag.StringVar(&OS, "OS", "windows", "Operating System (linux,windows)")
	flag.StringVar(&ARCH, "ARCH", "", "Architecture (32,64)")
	flag.StringVar(&GOROOT, "GOROOT", "", "Go root directory")
	flag.Parse()
	print("Usage")
	flag.PrintDefaults()

	print("BUILD STARTED")
	emptyln()
	// check OS 
	switch OS {
	case "windows":
		PATH_SEPARATOR = ";"
		PATH_SLASH = "\\"
	case "linux":
		PATH_SEPARATOR = ":"
		PATH_SLASH = "/"
	default:
		fail("Unkown os %v", OS)
	}

	// set up go build env
	setenv("GOROOT", GOROOT)
	path := getenv("PATH")
	setenv("PATH", GOROOT+PATH_SLASH+"bin"+PATH_SEPARATOR+path)

	// check ARCH
	switch ARCH {
	case "32":
		setenv("GOARCH", "386")
		VS_PLATFORM = "Win32"
	case "64":
		setenv("GOARCH", "amd64")
		VS_PLATFORM = "x64"
	default:
		fail("Unkown architecture %v", ARCH)

	}

	// display current go env variables
	execute("go", "env")
	print("Preparing staging area...")
	prepareStagingArea()
	success()
}

func done() {
	emptyln()
	print("BUILD SUCCEEDED")
}

func prepareStagingArea() {
	rm("pubsubsql")
	mkdir("./pubsubsql/bin")
	mkdir("./pubsubsql/lib")
	mkdir("./pubsubsql/samples")
}

func mkdir(path string) {
	err := os.MkdirAll(path, os.ModeDir|os.ModePerm)
	if err != nil {
		fail("Failed to create directory: %v error: %v", path, err)
	}
}

func cd(path string) {
	err := os.Chdir(path)
	if err != nil {
		fail("Failed to change directory: %v error: %v", path, err)
	}
}

func pwd() string {
	dir, err := os.Getwd()
	if err != nil {
		fail("Failed to get current directory: error: %v", err)
	}
	return dir
}

func rm(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fail("Fialed to remove path: %s error: %v", path, err)
	}
}

func execute(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		fail("Failed to execute command %v", err)
	}
}

func shellExt(file string) string {
	switch OS {
	case "linux":
		return file + ".sh"
	case "windows": 
		return file + ".bat"
	}
	fail("Invalid OS")
	return ""
}

func shellScript(file string) string {
	switch OS {
	case "linux":
		return "./" + shellExt(file) 
	case "windows": 
		return shellExt(file)
	}
	fail("Invalid OS")
	return ""
}

func shell(arg string) {
	println(arg)
	var cmd *exec.Cmd	
	switch OS {
	case "linux":
		cmd = exec.Command("/bin/sh", arg)		
	case "windows":
		cmd = exec.Command(arg)		
	default:
		fail("Invalid OS")
	}
	//
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		fail("Failed to execute shell %v", err)
	}
}

func setenv(key string, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		fail("Failed to set environment variable key:%v, value:% error:%v", key, value, err)
	}
}

func getenv(key string) string {
	return os.Getenv(key)
}

// copy

func copyFile(src string, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer dstFile.Close()
	
	if OS == "linux" {
		err = dstFile.Chmod(os.ModePerm)
		if err != nil {
			return
		}
	}
	
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return
	}
	
	err = os.Chtimes(dst, time.Now(), time.Now())	
	return err
}

func cp(src string, dst string) {
	err := copyFile(src, dst)
	if err != nil {
		fail("Failed to copy file %v", err)
	}
}

type fileCopy struct {
	from string
	to string
}

func (this *fileCopy) cp(file string) {
	cp(this.from + "/" + file, this.to + "/" + file)	
}

//

func open(path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		fail("Failed to open file %v error %v", path, err)
	}
	return file
}

func create(path string) *os.File {
	file, err := os.Create(path)
	if err != nil {
		fail("Failed to create file %v error %v", path, err)
	}
	err = os.Chtimes(path, time.Now(), time.Now())
	if err != nil {
		file.Close()
		fail("Failed to update time of file %v error %v", path, err)
	}
	return file
}

func getarchname() string {
	name := "pubsubsql-v" + VERSION + "-" + OS + "-"
	switch ARCH {
	case "32":
		name += "x86"
	case "64":
		name += "x64"
	}
	return name
}

func targz(archiveFile string, dir string) {
	// file
	file := create(archiveFile)
	defer file.Close()
	// gzip
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	// tar 
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	//	
	walk := func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo.Mode().IsDir() {
			return nil
		}
		if err != nil {
			fail("Failed to traverse directory structure %v", err)
		}
		print(path)
		fileToWrite := open(path)
		defer fileToWrite.Close()
		header, err := tar.FileInfoHeader(fileInfo, path)
		header.Name = path
		if err != nil {
			fail("Failed to create tar header from file info %v", err)
		}
		err = tarWriter.WriteHeader(header)
		if err != nil {
			fail("Failed to write tar header %v", err)
		}
		_, err = io.Copy(tarWriter, fileToWrite)
		if err != nil {
			fail("Failed to copy tar header %v", err)
		}
		return nil
	}
	//
	err := filepath.Walk(dir, walk)
	if err != nil {
		fail("Failed to traverse directory %v %v", dir, err)
	}
}

func dozip(archiveFile string, dir string) {
	// file	
	file := create(archiveFile)
	defer file.Close()
	// zip
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()
	//
	walk := func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			fail("Failed to traverse directory structure %v", err)
		}
		if fileInfo.Mode().IsDir() {
			return nil
		}
		print(path)
		fileToWrite := open(path)
		//
		var fileHeader *zip.FileHeader 
		fileHeader, err = zip.FileInfoHeader(fileInfo)
		if err != nil {
			fail("Failed to create file info header %v", err)
		}
		fileHeader.Name = path
		//		
		var w io.Writer
		w, err = zipWriter.CreateHeader(fileHeader)
		if err != nil {
			fail("Failed to create zip writer %v", err)
		}
		_, err = io.Copy(w, fileToWrite)
		if err != nil {
			fail("Failed to copy (zip writer) %v", err)
		}
		return nil
	}
	//
	err := filepath.Walk(dir, walk)
	if err != nil {
		fail("Failed to traverse directory %v %v", dir, err)
	}
}
