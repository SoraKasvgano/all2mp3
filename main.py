import os
import subprocess
import sys
import tkinter as tk
from tkinter import filedialog, messagebox, ttk
import threading
import shutil
import tempfile
from tkinterdnd2 import TkinterDnD, DND_FILES
import re

# 支持的输入格式
supported_formats = ['.mp4', '.m4a', '.m4s', '.aac', '.avi', '.flac', '.wav', '.ogg', '.wmv', '.mov', '.mpg', '.mpeg', '.webm', '.mkv']

class AudioConverter:
    def __init__(self, root):
        self.root = root
        self.root.title("音频格式转换器")
        self.root.geometry("600x800")
        self.root.resizable(True, True)
        
        # 初始化变量
        self.selected_files = []
        self.input_directory = ""
        
        # 检查FFmpeg
        self.ffmpeg_path = self.find_ffmpeg()
        
        # 创建GUI界面
        self.create_widgets()
        
        # 绑定拖拽事件
        self.bind_drop_events()
        
    def find_ffmpeg(self):
        """查找FFmpeg可执行文件"""
        # 先检查系统路径
        try:
            result = subprocess.run(['ffmpeg', '-version'], check=True, capture_output=True, text=True)
            return 'ffmpeg'
        except (subprocess.CalledProcessError, FileNotFoundError):
            pass
            
        # 检查当前目录
        ffmpeg_exe = os.path.join(os.getcwd(), 'ffmpeg.exe')
        if os.path.exists(ffmpeg_exe):
            return ffmpeg_exe
            
        # 检查子目录
        for subdir in ['ffmpeg', 'bin', 'tools']:
            ffmpeg_exe = os.path.join(os.getcwd(), subdir, 'ffmpeg.exe')
            if os.path.exists(ffmpeg_exe):
                return ffmpeg_exe
                
        return None
        
    def create_widgets(self):
        """创建GUI组件"""
        # 创建主框架
        main_frame = ttk.Frame(self.root, padding="20")
        main_frame.pack(fill=tk.BOTH, expand=True)
        
        # 标题标签
        title_label = ttk.Label(main_frame, text="音频格式转换器", font=('Arial', 16, 'bold'))
        title_label.pack(pady=10)
        
        # 输入文件/文件夹选择区域
        input_frame = ttk.LabelFrame(main_frame, text="输入文件/文件夹", padding="10")
        input_frame.pack(fill=tk.BOTH, expand=True, pady=10)
        
        # 选择按钮
        button_frame = ttk.Frame(input_frame)
        button_frame.pack(fill=tk.X, pady=5)
        
        select_file_btn = ttk.Button(button_frame, text="选择文件", command=self.select_files)
        select_file_btn.pack(side=tk.LEFT, padx=5)
        
        select_folder_btn = ttk.Button(button_frame, text="选择文件夹", command=self.select_folder)
        select_folder_btn.pack(side=tk.LEFT, padx=5)
        
        clear_btn = ttk.Button(button_frame, text="清空选择", command=self.clear_selection)
        clear_btn.pack(side=tk.RIGHT, padx=5)
        
        # 文件列表
        list_frame = ttk.Frame(input_frame)
        list_frame.pack(fill=tk.BOTH, expand=True, pady=5)
        
        scrollbar = ttk.Scrollbar(list_frame)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        
        self.file_listbox = tk.Listbox(list_frame, yscrollcommand=scrollbar.set, selectmode=tk.MULTIPLE)
        self.file_listbox.pack(fill=tk.BOTH, expand=True)
        scrollbar.config(command=self.file_listbox.yview)
        
        # 输出设置说明
        output_note = ttk.Label(main_frame, text="提示：转换后的MP3文件将保存在源文件相同目录下", foreground="blue")
        output_note.pack(pady=10)
        
        # 转换按钮
        self.convert_btn = ttk.Button(main_frame, text="开始转换", command=self.start_conversion, state=tk.DISABLED)
        self.convert_btn.pack(pady=20, fill=tk.X)
        
        # 进度条
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(main_frame, variable=self.progress_var, maximum=100)
        self.progress_bar.pack(fill=tk.X, pady=10)
        
        # 状态标签
        self.status_var = tk.StringVar(value="等待文件选择...")
        status_label = ttk.Label(main_frame, textvariable=self.status_var, foreground="blue", font=('Arial', 10, 'italic'))
        status_label.pack(pady=10)
        
        # FFmpeg状态
        if self.ffmpeg_path:
            ffmpeg_status = f"FFmpeg已找到: {self.ffmpeg_path}"
            ffmpeg_color = "green"
        else:
            ffmpeg_status = "警告: 未找到FFmpeg，请将ffmpeg.exe放置在程序目录"
            ffmpeg_color = "red"
        
        self.ffmpeg_label = ttk.Label(main_frame, text=ffmpeg_status, foreground=ffmpeg_color)
        self.ffmpeg_label.pack(pady=5)
        
    def bind_drop_events(self):
        """绑定拖拽事件到整个窗口"""
        try:
            # 使用tkinterdnd2库提供的拖拽事件绑定到整个窗口
            self.root.drop_target_register(DND_FILES)
            self.root.dnd_bind('<<Drop>>', self.on_drop)
            
            # 更新状态提示
            self.status_var.set("等待文件选择或拖拽文件...")
            
        except Exception as e:
            # 如果拖拽事件绑定失败，给出提示
            print(f"拖拽事件绑定失败: {e}")
            self.status_var.set("拖拽功能可能不可用，请使用'选择文件'或'选择文件夹'按钮")
        
    def on_drop(self, event):
        """拖拽放下事件"""
        try:
            # 使用tkinterdnd2的方式获取拖拽的文件路径
            file_paths = event.data
            if file_paths:
                print(f"原始拖拽数据: {file_paths}")
                # 处理Windows系统下的路径格式
                if isinstance(file_paths, str):
                    # 处理大括号包裹的路径 - 这是Windows系统下的特殊格式
                    # 例如: {D:/test/file1.m4s} {D:/test/file2.m4s}
                    import re
                    # 使用正则表达式提取大括号内的所有路径
                    # 匹配 {路径} 格式，路径可以包含空格和换行符
                    pattern = r'\{(.*?)\}'
                    file_paths = re.findall(pattern, file_paths, re.DOTALL)
                    
                    # 清理每个路径
                    file_paths = [path.strip() for path in file_paths if path.strip()]
                
                # 过滤掉可能的空路径
                file_paths = [path for path in file_paths if path]
                if file_paths:
                    self.add_files(file_paths)
        except Exception as e:
            print(f"拖拽放下事件错误: {e}")
            messagebox.showerror("错误", f"拖拽文件失败: {e}")
        
    def select_files(self):
        """选择文件"""
        file_types = [("音频/视频文件", "*.mp4 *.m4a *.m4s *.aac *.avi *.flac *.wav *.ogg *.wmv *.mov *.mpg *.mpeg *.webm *.mkv"), ("所有文件", "*.*")]
        file_paths = filedialog.askopenfilenames(title="选择文件", filetypes=file_types)
        if file_paths:
            self.add_files(file_paths)
            
    def select_folder(self):
        """选择文件夹"""
        folder_path = filedialog.askdirectory(title="选择文件夹")
        if folder_path:
            self.input_directory = folder_path
            # 遍历文件夹中的所有支持的文件
            files_to_add = []
            for root_dir, dirs, files in os.walk(folder_path):
                for file in files:
                    if os.path.splitext(file)[1].lower() in supported_formats:
                        files_to_add.append(os.path.join(root_dir, file))
            
            if files_to_add:
                self.add_files(files_to_add)
            else:
                messagebox.showinfo("提示", "所选文件夹中没有找到支持的文件格式")
                
    def add_files(self, file_paths):
        """添加文件到列表"""
        added_count = 0
        for file_path in file_paths:
            if file_path not in self.selected_files:
                # 检查文件格式
                ext = os.path.splitext(file_path)[1].lower()
                if ext in supported_formats:
                    self.selected_files.append(file_path)
                    self.file_listbox.insert(tk.END, os.path.basename(file_path))
                    added_count += 1
        
        # 更新按钮状态
        self.update_convert_button_state()
        
        if added_count > 0:
            self.status_var.set(f"已添加 {added_count} 个文件")
        
    def clear_selection(self):
        """清空选择"""
        self.selected_files = []
        self.file_listbox.delete(0, tk.END)
        self.update_convert_button_state()
        self.status_var.set("等待文件选择...")
        
    def browse_output_folder(self):
        """浏览输出文件夹"""
        folder_path = filedialog.askdirectory(title="选择输出文件夹")
        if folder_path:
            self.output_directory = folder_path
            self.output_path_var.set(folder_path)
            
    def update_convert_button_state(self):
        """更新转换按钮状态"""
        if self.selected_files and self.ffmpeg_path:
            self.convert_btn.config(state=tk.NORMAL)
        else:
            self.convert_btn.config(state=tk.DISABLED)
        
    def start_conversion(self):
        """开始转换"""
        if not self.selected_files:
            messagebox.showwarning("警告", "请先选择要转换的文件")
            return
            
        if not self.ffmpeg_path:
            messagebox.showerror("错误", "未找到FFmpeg，请将ffmpeg.exe放置在程序目录")
            return
            
        # 禁用界面控件
        self.convert_btn.config(state=tk.DISABLED)
        
        # 在新线程中执行转换
        threading.Thread(target=self.convert_files, daemon=True).start()
        
    def convert_files(self):
        """转换文件"""
        total_files = len(self.selected_files)
        success_count = 0
        
        for index, input_file in enumerate(self.selected_files):
            # 更新状态
            filename = os.path.basename(input_file)
            self.status_var.set(f"正在转换 ({index+1}/{total_files}): {filename}")
            
            # 生成输出文件名 - 使用源文件目录作为输出目录
            output_filename = os.path.splitext(filename)[0] + ".mp3"
            output_file = os.path.join(os.path.dirname(input_file), output_filename)
            
            # 转换文件
            if self.convert_to_mp3(input_file, output_file):
                success_count += 1
            
            # 更新进度
            progress = (index + 1) / total_files * 100
            self.progress_var.set(progress)
        
        # 转换完成
        self.status_var.set(f"转换完成！成功: {success_count}/{total_files}")
        self.convert_btn.config(state=tk.NORMAL)
        messagebox.showinfo("完成", f"转换完成！成功: {success_count}/{total_files}")
        
    def convert_to_mp3(self, input_file, output_file):
        """使用FFmpeg将音频文件转换为mp3"""
        try:
            cmd = [
                self.ffmpeg_path, '-y', '-i', input_file,
                '-vn', '-acodec', 'libmp3lame', '-ab', '192k', '-ar', '44100', '-ac', '2',
                output_file
            ]
            result = subprocess.run(cmd, check=True, capture_output=True, text=True, encoding='utf-8', errors='ignore')
            return True
        except subprocess.CalledProcessError as e:
            self.status_var.set(f"转换失败 {os.path.basename(input_file)}: {e.stderr}")
            return False
        except Exception as e:
            self.status_var.set(f"转换失败 {os.path.basename(input_file)}: {e}")
            return False

if __name__ == "__main__":
    root = TkinterDnD.Tk()
    app = AudioConverter(root)
    root.mainloop()