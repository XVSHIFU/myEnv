// Build with the Windows .NET Framework compiler; no installer runtime download.
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Drawing;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;
using System.Threading;
using System.Windows.Forms;
using Microsoft.Win32;

interface ISettings {
    string UserPath { get; set; }
    string Location { get; set; }
}

sealed class UserSettings : ISettings {
    const string AppKey = @"Software\Microsoft\Windows\CurrentVersion\Uninstall\myEnv.CLI";
    public string UserPath {
        get { using (var k = Registry.CurrentUser.OpenSubKey("Environment")) {
            return k == null ? "" : (string)k.GetValue("Path", "", RegistryValueOptions.DoNotExpandEnvironmentNames);
        } }
        set { using (var k = Registry.CurrentUser.CreateSubKey("Environment")) {
            var kind = k.GetValue("Path") == null ? RegistryValueKind.ExpandString : k.GetValueKind("Path");
            if (kind != RegistryValueKind.String && kind != RegistryValueKind.ExpandString) throw new IOException("用户 PATH 类型异常，请先检查环境变量。");
            k.SetValue("Path", value, kind);
        } }
    }
    public string Location {
        get { using (var k = Registry.CurrentUser.OpenSubKey(AppKey)) { return k == null ? "" : (string)k.GetValue("InstallLocation", ""); } }
        set {
            if (value == "") { Registry.CurrentUser.DeleteSubKey(AppKey, false); return; }
            using (var k = Registry.CurrentUser.CreateSubKey(AppKey)) {
                k.SetValue("InstallLocation", value);
                k.SetValue("DisplayName", "myEnv CLI（当前用户）");
                k.SetValue("DisplayVersion", Setup.Version);
                k.SetValue("UninstallString", "\"" + Path.Combine(value, "myenv-setup.exe") + "\" --uninstall");
                k.SetValue("NoModify", 1, RegistryValueKind.DWord);
                k.SetValue("NoRepair", 1, RegistryValueKind.DWord);
            }
        }
    }
}

sealed class MemorySettings : ISettings {
    public string UserPath { get; set; }
    string location;
    public bool FailNextRegistration;
    public string Location { get {return location;} set {if(FailNextRegistration){FailNextRegistration=false;throw new IOException("injected registration failure");}location=value;} }
    public MemorySettings() { UserPath = ""; Location = ""; }
}

static class Install {
    const string Marker = "myenv-install.txt";
    static readonly StringComparer Paths = StringComparer.OrdinalIgnoreCase;
    public static string Normalize(string path) {
        return Path.GetFullPath(Environment.ExpandEnvironmentVariables(path.Trim().Trim('"'))).TrimEnd(Path.DirectorySeparatorChar);
    }
    public static string Validate(string path) {
        string root = Normalize(path);
        if (!Path.IsPathRooted(path) || root.Length < 4 || root.Contains(";") || root.StartsWith(@"\\"))
            throw new IOException("请选择本机的独立绝对目录，不能包含分号或使用磁盘根目录。");
        for (var d = new DirectoryInfo(root); d != null; d = d.Parent)
            if (d.Exists && (d.Attributes & FileAttributes.ReparsePoint) != 0) throw new IOException("安装路径不能经过链接或 junction：" + d.FullName);
        return root;
    }
    static bool Same(string a, string b) {
        if (String.IsNullOrWhiteSpace(a) || String.IsNullOrWhiteSpace(b)) return false;
        try { return Paths.Equals(Normalize(a), Normalize(b)); } catch { return false; }
    }
    public static bool ContainsPath(string value, string root) { return value.Split(';').Any(p => Same(p, root)); }
    public static string AddPath(string value, string root) {
        if (ContainsPath(value, root)) return value;
        return value + (value.Length == 0 || value.EndsWith(";") ? "" : ";") + root;
    }
    public static string RemovePath(string value, string root) {
        bool removed=false;
        return String.Join(";", value.Split(';').Where(p => {if(!removed && Same(p,root)){removed=true;return false;}return true;}).ToArray());
    }
    public static string Digest(byte[] data) { using (var h = SHA256.Create()) { return BitConverter.ToString(h.ComputeHash(data)).Replace("-", "").ToLowerInvariant(); } }
    static byte[] ReadOptional(string path) { return File.Exists(path) ? File.ReadAllBytes(path) : null; }
    static void WriteAtomic(string path, byte[] data) {
        string temp=path+"."+Guid.NewGuid().ToString("N")+".tmp";
        try {
            using(var stream=new FileStream(temp,FileMode.CreateNew,FileAccess.Write,FileShare.None)) {stream.Write(data,0,data.Length);stream.Flush(true);}
            if(File.Exists(path)) File.Replace(temp,path,null); else File.Move(temp,path);
        }finally{if(File.Exists(temp))File.Delete(temp);}
    }
    static void Restore(string path, byte[] data) { if (data == null) { if (File.Exists(path)) File.Delete(path); } else if (!File.Exists(path) || Digest(File.ReadAllBytes(path)) != Digest(data)) WriteAtomic(path, data); }
    static string[] Owned(string root) {
        string path = Path.Combine(root, Marker);
        if (!File.Exists(path) || (File.GetAttributes(path)&FileAttributes.ReparsePoint)!=0) throw new IOException("目录中没有有效 myEnv 安装记录；不会覆盖或删除不明文件。");
        var lines = File.ReadAllLines(path, Encoding.UTF8);
        if (lines.Length != 5 || lines[0] != "myEnv-user-install-v1" || !Same(lines[1], root) || (lines[2] != "yes" && lines[2] != "no"))
            throw new IOException("myEnv 安装记录无效。");
        var names = new [] { "myenv.exe", "myenv-setup.exe" };
        for (int i = 0; i < names.Length; i++) {
            string file = Path.Combine(root, names[i]);
            if (!File.Exists(file) || (File.GetAttributes(file) & FileAttributes.ReparsePoint) != 0 || Digest(File.ReadAllBytes(file)) != lines[i+3])
                throw new IOException("已安装文件缺失或被修改，请保留现场后手动检查：" + file);
        }
        return lines;
    }
    public static List<string> Conflicts(string paths, string root) {
        var found = new List<string>();
        foreach (var part in paths.Split(';')) {
            if (String.IsNullOrWhiteSpace(part) || Same(part, root)) continue;
            string dir;
            try { dir = Normalize(part); } catch { continue; }
            foreach (var ext in new [] { ".exe", ".com", ".cmd", ".bat", ".ps1" }) {
                string file = Path.Combine(dir, "myenv"+ext);
                if (File.Exists(file) && !found.Contains(file)) found.Add(file);
            }
        }
        return found;
    }
    public static void Apply(string path, bool addPath, ISettings settings, byte[] cli, byte[] setup) {
        string root = Validate(path);
        if (!String.IsNullOrEmpty(settings.Location) && !Same(settings.Location,root)) throw new IOException("已在其他目录安装，请先卸载原安装："+settings.Location);
        string[] previous = null;
        if (Directory.Exists(root) && Directory.EnumerateFileSystemEntries(root).Any()) previous = Owned(root);
        // Never overwrite a second process's installation/update/uninstall.
        Directory.CreateDirectory(root);
        string exe = Path.Combine(root,"myenv.exe"), helper=Path.Combine(root,"myenv-setup.exe"), marker=Path.Combine(root,Marker);
        byte[] oldExe=ReadOptional(exe), oldHelper=ReadOptional(helper), oldMarker=ReadOptional(marker);
        string oldPath=settings.UserPath, oldLocation=settings.Location;
        bool ownedPath = (previous != null && previous[2] == "yes") || (addPath && !ContainsPath(oldPath,root));
        string updated = addPath ? AddPath(oldPath,root) : oldPath;
        try {
            // An in-use executable fails before any PATH change. Keep backups in memory
            // until the complete local transaction succeeds; installers are low frequency.
            WriteAtomic(exe,cli);
            WriteAtomic(helper,setup);
            WriteAtomic(marker,Encoding.UTF8.GetBytes(String.Join("\n",new [] {"myEnv-user-install-v1",root,ownedPath?"yes":"no",Digest(cli),Digest(setup)})+"\n"));
            if (settings.UserPath != oldPath) throw new IOException("安装期间用户 PATH 已改变，请重试。");
            if (updated != oldPath) settings.UserPath=updated;
            settings.Location=root;
        } catch (Exception original) {
            var failures=new List<string>();
            try { Restore(exe,oldExe); Restore(helper,oldHelper); Restore(marker,oldMarker); } catch(Exception e) {failures.Add(e.Message);}
            try { if (settings.UserPath == updated && updated != oldPath) settings.UserPath=oldPath; settings.Location=oldLocation; } catch(Exception e) {failures.Add(e.Message);}
            throw new IOException(original.Message+(failures.Count==0?"\n此前的安装和 PATH 已保留。":"\n回退未全部完成，请检查："+String.Join("；",failures.ToArray())), original);
        }
    }
    public static void Remove(string path, ISettings settings) {
        string root=Validate(path);
        if (!Same(settings.Location,root)) throw new IOException("卸载目录与当前用户的安装登记不一致。");
        string[] previous=Owned(root);
        string exe=Path.Combine(root,"myenv.exe"), helper=Path.Combine(root,"myenv-setup.exe"), marker=Path.Combine(root,Marker);
        byte[] oldExe=File.ReadAllBytes(exe),oldHelper=File.ReadAllBytes(helper),oldMarker=File.ReadAllBytes(marker);
        string oldPath=settings.UserPath;
        string updated=previous[2]=="yes" ? RemovePath(oldPath,root) : oldPath;
        try {
            File.Delete(exe); File.Delete(helper); File.Delete(marker);
            if (settings.UserPath!=oldPath) throw new IOException("卸载期间 PATH 已改变，请重试。");
            if (updated!=oldPath) settings.UserPath=updated;
            settings.Location="";
        } catch {
            Restore(exe,oldExe); Restore(helper,oldHelper); Restore(marker,oldMarker);
            if (settings.UserPath==updated && updated!=oldPath) settings.UserPath=oldPath;
            settings.Location=root;
            throw;
        }
        // Never recursively delete: user-added files and runtime/project data survive.
        if (!Directory.EnumerateFileSystemEntries(root).Any()) Directory.Delete(root,false);
    }
}

static class Setup {
    public static string Version { get { using (var s=Assembly.GetExecutingAssembly().GetManifestResourceStream("version.txt")) using (var r=new StreamReader(s)) return r.ReadToEnd().Trim(); } }
    static byte[] Payload() {
        using (var s=Assembly.GetExecutingAssembly().GetManifestResourceStream("myenv.exe")) using(var m=new MemoryStream()) {s.CopyTo(m);return m.ToArray();}
    }
    [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    static extern IntPtr SendMessageTimeout(IntPtr h,uint msg,UIntPtr w,string text,uint flags,uint timeout,out UIntPtr result);
    static void Notify() {UIntPtr result; SendMessageTimeout(new IntPtr(0xffff),0x1a,UIntPtr.Zero,"Environment",2,2000,out result);}
    static string Self { get {return Assembly.GetExecutingAssembly().Location;} }
    static void SpawnWorker(string root) {
        string temp=Path.Combine(Path.GetTempPath(),"myenv-uninstall-"+Guid.NewGuid().ToString("N")+".exe");
        File.Copy(Self,temp);
        Process.Start(new ProcessStartInfo(temp,"--remove-worker "+Process.GetCurrentProcess().Id+" \""+root+"\"") {UseShellExecute=false,CreateNoWindow=true});
    }
    [STAThread]
    static int Main(string[] args) {
        if (args.Length==2 && args[0]=="--self-test") return SelfTest(args[1]);
        Application.EnableVisualStyles(); Application.SetCompatibleTextRenderingDefault(false);
        try {
            if(Environment.OSVersion.Version.Major<10)throw new IOException("myEnv 需要 Windows 10 / Server 2016 或更新版本。");
            var settings=new UserSettings();
            if (args.Length==3 && args[0]=="--remove-worker") {
                try {using(var parent=Process.GetProcessById(Int32.Parse(args[1]))) {if(!parent.WaitForExit(15000)) throw new IOException("安装向导尚未退出，请稍后重试。");}} catch(ArgumentException) {}
                using (var mutex=new Mutex(false,@"Local\myEnv.CLI.Install")) {
                    if (!mutex.WaitOne(0)) throw new IOException("另一个 myEnv 安装操作正在进行。");
                    try {Install.Remove(args[2],settings);} finally {mutex.ReleaseMutex();}
                }
                Notify(); MessageBox.Show("myEnv 命令已卸载；项目与运行时数据已保留。\n请重新打开终端。临时卸载助手可由系统临时文件清理移除。","myEnv"); return 0;
            }
            bool uninstall=args.Length==1 && args[0]=="--uninstall";
            bool preview=args.Length==2 && args[0]=="--preview";
            if (args.Length!=0 && !uninstall && !preview) throw new IOException("不支持的安装参数。");
            using (var form=new Form()) {
                form.Text="myEnv CLI 安装向导 · "+Version; form.ClientSize=new Size(640,365); form.StartPosition=FormStartPosition.CenterScreen;
                form.Font=new Font("Microsoft YaHei UI",9F);
                form.FormBorderStyle=FormBorderStyle.FixedDialog; form.MaximizeBox=false;
                var intro=new Label {Left=24,Top=22,Width=590,Height=74,Text="安装后可直接输入 myenv，无需寻找 exe。\n仅安装到当前用户，不需要管理员权限。\n这是开发版；中文帮助已接入，性能验收仍有开放项。"};
                var pathLabel=new Label{Left=24,Top=110,Width=570,Text="安装目录（建议保持默认，升级沿用原目录）："};
                var path=new TextBox{Left=24,Top=139,Width=495,Text=String.IsNullOrEmpty(settings.Location)?Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),"Programs","myEnv"):settings.Location};
                var browse=new Button{Left=530,Top=137,Width=82,Text="浏览…"};
                browse.Click+=(sender,e)=>{using(var picker=new FolderBrowserDialog()){picker.Description="选择独立的 myEnv 安装目录";picker.SelectedPath=path.Text;if(picker.ShowDialog()==DialogResult.OK)path.Text=picker.SelectedPath;}};
                var add=new CheckBox{Left=24,Top=183,Width=565,Checked=true,Text="加入当前用户 PATH（安装后重新打开终端即可使用）"};
                var hint=new Label{Left=24,Top=220,Width=590,Height=45,Text="不会替换系统 Node/Python，不安装项目依赖。\n卸载保留项目、环境和缓存；仅移除本向导添加的 PATH 项。"};
                var remove=new Button{Left=24,Top=293,Width=108,Height=34,Text="卸载",Enabled=!String.IsNullOrEmpty(settings.Location)};
                var install=new Button{Left=390,Top=293,Width=108,Height=34,Text="安装 / 升级"};
                var close=new Button{Left=510,Top=293,Width=102,Height=34,Text="关闭"};
                close.Click+=(sender,e)=>form.Close();
                remove.Click+=(sender,e)=>{if(MessageBox.Show("卸载 myEnv 命令并保留项目与环境数据？", "myEnv", MessageBoxButtons.OKCancel)==DialogResult.OK){SpawnWorker(settings.Location);form.Close();}};
                install.Click+=(sender,e)=>{
                    install.Enabled=false;
                    try {
                        string root=Install.Validate(path.Text);
                        var conflicts=Install.Conflicts((Environment.GetEnvironmentVariable("PATH")??"")+";"+settings.UserPath,root);
                        if(add.Checked && conflicts.Count>0) throw new IOException("发现同名命令，请先处理冲突，或取消加入 PATH 后使用绝对路径：\n"+String.Join("\n",conflicts.ToArray()));
                        using(var mutex=new Mutex(false,@"Local\myEnv.CLI.Install")) {
                            if(!mutex.WaitOne(0))throw new IOException("另一个 myEnv 安装操作正在进行。");
                            try{Install.Apply(root,add.Checked,settings,Payload(),File.ReadAllBytes(Self));}finally{mutex.ReleaseMutex();}
                        }
                        Notify();
                        MessageBox.Show(add.Checked?"安装完成。请重新打开终端，运行：\nmyenv --version\nmyenv --help\n\n若终端仍未识别命令，请退出终端应用后重开。":"安装完成（未添加 PATH）。\n请通过安装目录下的 myenv.exe 运行。","myEnv");
                        form.Close();
                    }catch(Exception ex){MessageBox.Show(ex.Message,"安装未完成",MessageBoxButtons.OK,MessageBoxIcon.Error);}finally{install.Enabled=true;}
                };
                form.Controls.AddRange(new Control[]{intro,pathLabel,path,browse,add,hint,remove,install,close});
                if(uninstall){install.Enabled=false;path.Enabled=false;browse.Enabled=false;add.Enabled=false;}
                if(preview){form.StartPosition=FormStartPosition.Manual;form.Location=new Point(-32000,-32000);form.Show();Application.DoEvents();using(var bitmap=new Bitmap(form.Width,form.Height)){form.DrawToBitmap(bitmap,new Rectangle(0,0,form.Width,form.Height));bitmap.Save(args[1],System.Drawing.Imaging.ImageFormat.Png);}form.Close();return 0;}
                Application.Run(form);
            }
            return 0;
        }catch(Exception e){MessageBox.Show(e.Message,"myEnv 安装向导",MessageBoxButtons.OK,MessageBoxIcon.Error);return 1;}
    }
    static void Check(bool value,string name) {if(!value)throw new Exception(name);}
    static int SelfTest(string directory) {
        // No registry, PATH mutation, message boxes or child execution in this mode.
        string root=Install.Validate(directory);
        try {
            if(Directory.Exists(root))throw new IOException("Self-test requires a fresh directory.");
            Directory.CreateDirectory(root);
            var state=new MemorySettings{UserPath=@"%USERPROFILE%\bin;C:\unrelated"};
            string app=Path.Combine(root,"中文 app");byte[] cli=Payload(), setup=File.ReadAllBytes(Self);
            Check(cli.Length>100000 && cli[0]=='M' && cli[1]=='Z',"payload");
            Install.Apply(app,true,state,cli,setup);Check(Install.ContainsPath(state.UserPath,app),"PATH add");
            string once=state.UserPath;
            Install.Apply(app,true,state,cli,setup);Check(state.UserPath==once,"idempotent upgrade");
            state.UserPath+=@";C:\user-added";
            File.WriteAllText(Path.Combine(app,"keep.txt"),"user content");
            Install.Remove(app,state);
            Check(state.UserPath==@"%USERPROFILE%\bin;C:\unrelated;C:\user-added","preserve PATH");
            Check(File.Exists(Path.Combine(app,"keep.txt")) && !File.Exists(Path.Combine(app,"myenv.exe")),"preserve user files");
            string pre=Path.Combine(root,"preexisting-path");state.UserPath=pre;
            Install.Apply(pre,true,state,cli,setup);Install.Remove(pre,state);Check(state.UserPath==pre,"unowned PATH preserved");
            string collision=Path.Combine(root,"collision");Directory.CreateDirectory(collision);File.WriteAllText(Path.Combine(collision,"myenv.exe"),"unrelated");
            bool refused=false;try{Install.Apply(collision,true,state,cli,setup);}catch(IOException){refused=true;}
            Check(refused && File.ReadAllText(Path.Combine(collision,"myenv.exe"))=="unrelated","foreign file protection");
            Check(Install.Conflicts(collision,pre).Count==1,"conflict detection");
            Check(Install.RemovePath(pre+";"+pre,pre)==pre,"remove only one owned PATH entry");
            string rollback=Path.Combine(root,"rollback");string originalPath=state.UserPath;state.FailNextRegistration=true;
            refused=false;try{Install.Apply(rollback,true,state,cli,setup);}catch(IOException){refused=true;}
            Check(refused && state.UserPath==originalPath && state.Location=="" && !File.Exists(Path.Combine(rollback,"myenv.exe")),"registration failure rollback");
            Install.Apply(rollback,true,state,cli,setup);
            using(var locked=new FileStream(Path.Combine(rollback,"myenv.exe"),FileMode.Open,FileAccess.Read,FileShare.None)) {
                refused=false;try{Install.Remove(rollback,state);}catch(IOException){refused=true;}
                Check(refused && Install.ContainsPath(state.UserPath,rollback),"locked executable keeps PATH");
            }
            File.WriteAllText(Path.Combine(rollback,"myenv.exe"),"modified");
            refused=false;try{Install.Remove(rollback,state);}catch(IOException){refused=true;}
            Check(refused && File.ReadAllText(Path.Combine(rollback,"myenv.exe"))=="modified","modified binary retained");
            File.WriteAllText(Path.Combine(root,"result.txt"),"PASS: payload, install, upgrade, PATH ownership, uninstall, retained data, conflicts, registration rollback, locked/modified binary protection\n",Encoding.UTF8);
            return 0;
        }catch(Exception e){if(Directory.Exists(root))File.WriteAllText(Path.Combine(root,"failure.txt"),e.ToString());return 1;}
    }
}
