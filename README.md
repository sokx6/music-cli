# music-cli - 一个简单的命令行音乐播放器

这是一个用 Go 编写的简单命令行音乐播放器。

## 功能
- 播放音乐 (
- 显示双语歌词
- 进度条显示
- 显示逐字歌词
- 暂停和继续
- 下一首
- 播放文件夹内所有文件
- 分页显示当前目录
- 播放当前目录音乐
- 递归遍历当前目录和子目录播放音乐

## 支持的格式
- mp3
- flac
- wav

## 前提

- 已安装 Go（版本 1.25.3）

## 构建与运行

```powershell
# 克隆仓库
git clone https://github.com/sokx6/music-cli.git

# 进入目录
cd music-cli

# 构建可执行文件
go build

# 运行可执行文件
./music-cli

# 或者
music-cli
```

