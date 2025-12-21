package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ================= 🌍 多语言包 (i18n) =================
var LangMap = map[string]map[string]string{
	"zh": {
		"LoginTitle": "登录 ExpiryGuard", "User": "用户名", "Pass": "密码", "BtnLogin": "登录系统",
		"NoAccount": "还没有账号？", "SignUp": "立即注册", "RegTitle": "注册新账号", "BtnReg": "确认注册",
		"ForceTitle": "安全警告：需修改密码", "ForceMsg": "管理员已重置您的密码，请设置新密码以继续。",
		"Dashboard": "控制台", "Profile": "个人设置", "Admin": "系统管理", "Logout": "退出",
		"Total": "总资产", "Expiring": "30天内到期", "Urgent": "急需处理 (7天)",
		"MyAssets": "资产列表", "Export": "导出 CSV", "Import": "导入", "Backup": "下载数据库备份",
		"Search": "搜索分类、名称、到期日...",
		"Cat": "分类", "Name": "名称", "Date": "到期日", "Action": "操作",
		"Add": "添加资产", "Details": "详细信息 (账号/密码/备注/密钥)", "View": "查看", "Edit": "编辑", "Del": "删除",
		"Save": "保存设置", "Cancel": "取消", "Close": "关闭", "Copy": "复制内容", "Upload": "确认导入",
		"NotifySettings": "通知设置 (绑定后自动开启登录验证)", "ChatID": "Telegram ID", "Email": "接收邮箱",
		"Timezone": "显示时区", "NotifyTime": "推送时间", "GetID": "获取 ID", "TestBtn": "发送测试消息",
		"BotStatus": "机器人运行正常", "OpenBot": "打开机器人",
		"Security": "安全设置", "NewPass": "新密码", "ChangePass": "修改密码",
		"GlobalSet": "全局配置", "UserMgmt": "用户管理", "CatMgmt": "分类管理",
		"Role": "角色", "ResetPwd": "重置密码", "Reset2FA": "重置2FA", "DelUser": "删除用户",
		"Msg_2FA": "安全验证", "Msg_CodeSent": "验证码已发送至您的设备", "Code": "输入验证码", "Verify": "验证并登录",
		"SelectMethod": "选择验证方式", "ViaTG": "Telegram 发送", "ViaEmail": "邮件发送",
		"Ph_BotUser": "例如: MyBot (不带@)", "Ph_BotToken": "例如: 123456:ABC-DEF...",
		"Ph_Host": "例如: smtp.qq.com", "Ph_Port": "例如: 465", "Ph_User": "邮箱账号", "Ph_Pass": "授权码/密码",
		"Ph_Cat": "输入或选择分类 (如: 服务器)",
		"Msg_Add": "资产添加成功", "Msg_Upd": "资产更新成功", "Msg_Del": "删除成功", 
		"Msg_Saved": "配置已保存", "Msg_Sent": "发送成功", "Msg_Fail": "发送失败",
		"Msg_Reset": "密码已重置为 123456", "Msg_2FACleared": "2FA 已清除",
		"Msg_UserDel": "用户已删除", "Msg_PwdChanged": "密码修改成功，请重新登录",
		"Msg_RegSuccess": "注册成功，请登录", "Msg_LoginSuccess": "欢迎回来，登录成功", "Msg_Imported": "导入成功",
		"Msg_LoginErr": "登录失败：用户名或密码错误", "Msg_CodeErr": "验证码错误或已过期",
		"ConfirmDel": "确定要删除吗？此操作不可恢复。", "ConfirmReset": "确定要重置吗？",
		"Head_ID": "ID", "Head_User": "用户名", "Head_Role": "角色", "Head_Info": "资产/绑定", "Head_IP": "最后登录 IP", "Head_Action": "操作",
		"TZ_Shanghai": "🇨🇳 中国 (北京/上海)", "TZ_Tokyo": "🇯🇵 日本 (东京)", "TZ_Seoul": "🇰🇷 韩国 (首尔)",
		"TZ_Singapore": "🇸🇬 新加坡", "TZ_London": "🇬🇧 英国 (伦敦)", "TZ_Berlin": "🇩🇪 德国 (柏林)",
		"TZ_NY": "🇺🇸 美国 (纽约)", "TZ_LA": "🇺🇸 美国 (洛杉矶)", "TZ_UTC": "🌐 UTC 标准时间",
		"Notify_Title": "续费提醒", "Notify_Test": "测试通知", "Notify_Login": "登录验证",
		"Notify_Body_Login": "您的登录验证码是：", "Notify_Body_IP": "请求来源 IP：",
		"Notify_Body_Test": "这是一条来自 ExpiryGuard 的测试消息，您的通知服务工作正常。",
		"Simulate_Title": "模拟报警测试", "Simulate_Desc": "发送一条模拟的到期提醒给当前管理员，用于预览通知效果。", "Simulate_Btn": "发送模拟报警",
		"Sim_Domain": "域名", "Sim_Server": "服务器", "Sim_DaysLeft": "天后到期", "Sim_Expired": "已过期!",
		"Import_Title": "批量导入资产 (CSV)", "Import_Desc": "请上传 CSV 文件，格式需为: 分类,名称,日期(YYYY-MM-DD),详情",
	},
	"en": {
		"LoginTitle": "Login ExpiryGuard", "User": "Username", "Pass": "Password", "BtnLogin": "Sign In",
		"NoAccount": "No account?", "SignUp": "Sign up", "RegTitle": "Create Account", "BtnReg": "Register",
		"ForceTitle": "Security Alert: Reset Required", "ForceMsg": "Admin reset your password. Please set a new one.",
		"Dashboard": "Dashboard", "Profile": "Settings", "Admin": "Admin", "Logout": "Logout",
		"Total": "Total Assets", "Expiring": "Expiring (30d)", "Urgent": "Urgent (7d)",
		"MyAssets": "My Assets", "Export": "Export", "Import": "Import", "Backup": "Download DB Backup",
		"Search": "Search Category, Name, Date...", 
		"Cat": "Category", "Name": "Name", "Date": "Due Date", "Action": "Action",
		"Add": "Add Asset", "Details": "Details (Account/Pwd/Notes/Keys)", "View": "View", "Edit": "Edit", "Del": "Delete",
		"Save": "Save Changes", "Cancel": "Cancel", "Close": "Close", "Copy": "Copy Info", "Upload": "Upload CSV",
		"NotifySettings": "Notifications (Enables 2FA on Login)", "ChatID": "Telegram ID", "Email": "Email",
		"Timezone": "Timezone", "NotifyTime": "Notify Time", "GetID": "Get ID", "TestBtn": "Send Test Alert",
		"BotStatus": "Bot Service Active", "OpenBot": "Open Bot",
		"Security": "Security", "NewPass": "New Password", "ChangePass": "Change Password",
		"GlobalSet": "Global Settings", "UserMgmt": "Users",
		"Role": "Role", "ResetPwd": "Reset Pwd", "Reset2FA": "Reset 2FA", "DelUser": "Delete",
		"Msg_2FA": "2FA Required", "Msg_CodeSent": "Verification code sent", "Code": "Enter Code", "Verify": "Verify & Login",
		"SelectMethod": "Select Method", "ViaTG": "Via Telegram", "ViaEmail": "Via Email",
		"Ph_BotUser": "e.g., MyBot (no @)", "Ph_BotToken": "e.g., 123456:ABC-DEF...",
		"Ph_Host": "e.g., smtp.gmail.com", "Ph_Port": "e.g., 465", "Ph_User": "Email User", "Ph_Pass": "App Password",
		"Ph_Cat": "Type or select category",
		"Msg_Add": "Added successfully", "Msg_Upd": "Updated successfully", "Msg_Del": "Deleted", 
		"Msg_Saved": "Settings saved", "Msg_Sent": "Sent successfully", "Msg_Fail": "Failed to send",
		"Msg_Reset": "Password reset to 123456", "Msg_2FACleared": "2FA Cleared",
		"Msg_UserDel": "User Deleted", "Msg_PwdChanged": "Password Changed, Please Login",
		"Msg_RegSuccess": "Registration successful", "Msg_LoginSuccess": "Login successful", "Msg_Imported": "Import Successful",
		"Msg_LoginErr": "Login failed: Invalid credentials", "Msg_CodeErr": "Invalid or expired code",
		"ConfirmDel": "Are you sure you want to delete?", "ConfirmReset": "Are you sure?",
		"Head_ID": "ID", "Head_User": "User", "Head_Role": "Role", "Head_Info": "Assets/Bind", "Head_IP": "Last IP", "Head_Action": "Action",
		"TZ_Shanghai": "🇨🇳 China (Beijing)", "TZ_Tokyo": "🇯🇵 Japan (Tokyo)", "TZ_Seoul": "🇰🇷 Korea (Seoul)",
		"TZ_Singapore": "🇸🇬 Singapore", "TZ_London": "🇬🇧 UK (London)", "TZ_Berlin": "🇩🇪 Germany (Berlin)",
		"TZ_NY": "🇺🇸 USA (New York)", "TZ_LA": "🇺🇸 USA (Los Angeles)", "TZ_UTC": "🌐 UTC",
		"Notify_Title": "Expiry Alert", "Notify_Test": "Test Notification", "Notify_Login": "Login Verification",
		"Notify_Body_Login": "Your verification code:", "Notify_Body_IP": "Request IP:",
		"Notify_Body_Test": "This is a test message from ExpiryGuard. Your notification service is working.",
		"Simulate_Title": "Simulation Test", "Simulate_Desc": "Send a fake alert to the current admin to preview the notification style.", "Simulate_Btn": "Send Simulation",
		"Sim_Domain": "Domain", "Sim_Server": "Server", "Sim_DaysLeft": "days left", "Sim_Expired": "EXPIRED!",
		"Import_Title": "Batch Import (CSV)", "Import_Desc": "Upload CSV file. Format: Category,Name,Date(YYYY-MM-DD),Detail",
	},
	"ja": {
		"LoginTitle": "ExpiryGuard ログイン", "User": "ユーザー名", "Pass": "パスワード", "BtnLogin": "ログイン",
		"NoAccount": "アカウントをお持ちでないですか？", "SignUp": "登録", "RegTitle": "アカウント作成", "BtnReg": "登録",
		"ForceTitle": "パスワード変更が必要です", "ForceMsg": "管理者がパスワードをリセットしました。新しいパスワードを設定してください。",
		"Dashboard": "ダッシュボード", "Profile": "設定", "Admin": "管理", "Logout": "ログアウト",
		"Total": "総資産", "Expiring": "期限切れ (30日)", "Urgent": "緊急 (7日)",
		"MyAssets": "資産リスト", "Export": "出力", "Import": "取込", "Backup": "DBバックアップ",
		"Search": "検索（分類、名称、日付）...",
		"Cat": "カテゴリ", "Name": "名称", "Date": "有効期限", "Action": "操作",
		"Add": "追加", "Details": "詳細 (ID/PW/備考)", "View": "詳細", "Edit": "編集", "Del": "削除",
		"Save": "保存", "Cancel": "キャンセル", "Close": "閉じる", "Copy": "コピー", "Upload": "アップロード",
		"NotifySettings": "通知設定 (ログイン2FA有効化)", "ChatID": "Telegram ID", "Email": "メール",
		"Timezone": "タイムゾーン", "NotifyTime": "通知時間", "GetID": "ID取得", "TestBtn": "テスト送信",
		"BotStatus": "Bot稼働中", "OpenBot": "Botを開く",
		"Security": "セキュリティ", "NewPass": "新しいパスワード", "ChangePass": "変更",
		"Msg_2FA": "2段階認証が必要です", "Msg_CodeSent": "コード送信完了", "Code": "認証コード", "Verify": "認証",
		"SelectMethod": "認証方法を選択", "ViaTG": "Telegramで送信", "ViaEmail": "メールで送信",
		"GlobalSet": "システム設定", "UserMgmt": "ユーザー管理",
		"Role": "権限", "ResetPwd": "PWリセット", "Reset2FA": "2FAリセット", "DelUser": "削除",
		"Ph_BotUser": "例: MyBot (@なし)", "Ph_BotToken": "例: 123456:ABC...",
		"Ph_Host": "例: smtp.gmail.com", "Ph_Port": "例: 465", "Ph_User": "メールアドレス", "Ph_Pass": "パスワード",
		"Ph_Cat": "カテゴリを入力または選択",
		"Msg_Add": "追加しました", "Msg_Upd": "更新しました", "Msg_Del": "削除しました", 
		"Msg_Saved": "保存しました", "Msg_Sent": "送信しました", "Msg_Fail": "送信失敗",
		"Msg_Reset": "PWを123456にリセット", "Msg_2FACleared": "2FAを解除しました",
		"Msg_UserDel": "ユーザーを削除しました", "Msg_PwdChanged": "パスワードを変更しました",
		"Msg_RegSuccess": "登録完了、ログインしてください", "Msg_LoginSuccess": "おかえりなさい", "Msg_Imported": "インポート成功",
		"Msg_LoginErr": "ログイン失敗：ユーザー名またはパスワードが間違っています", "Msg_CodeErr": "コードが無効または期限切れです",
		"ConfirmDel": "削除してもよろしいですか？", "ConfirmReset": "リセットしますか？",
		"Head_ID": "ID", "Head_User": "ユーザー", "Head_Role": "権限", "Head_Info": "資産/連携", "Head_IP": "最終IP", "Head_Action": "操作",
		"TZ_Shanghai": "🇨🇳 中国 (北京)", "TZ_Tokyo": "🇯🇵 日本 (東京)", "TZ_Seoul": "🇰🇷 韓国 (ソウル)",
		"TZ_Singapore": "🇸🇬 シンガポール", "TZ_London": "🇬🇧 英国 (ロンドン)", "TZ_Berlin": "🇩🇪 ドイツ (ベルリン)",
		"TZ_NY": "🇺🇸 米国 (NY)", "TZ_LA": "🇺🇸 米国 (LA)", "TZ_UTC": "🌐 UTC",
		"Notify_Title": "期限切れ通知", "Notify_Test": "テスト通知", "Notify_Login": "ログイン認証",
		"Notify_Body_Login": "認証コード：", "Notify_Body_IP": "IP：",
		"Notify_Body_Test": "これはExpiryGuardからのテストメッセージです。",
		"Simulate_Title": "アラートシミュレーション", "Simulate_Desc": "現在の管理者に模擬アラートを送信して、通知スタイルを確認します。", "Simulate_Btn": "模擬送信",
		"Sim_Domain": "ドメイン", "Sim_Server": "サーバー", "Sim_DaysLeft": "日残り", "Sim_Expired": "期限切れ!",
		"Import_Title": "一括インポート (CSV)", "Import_Desc": "CSVをアップロードしてください。形式: 分類,名称,日付(YYYY-MM-DD),詳細",
	},
	"ko": {
		"LoginTitle": "ExpiryGuard 로그인", "User": "사용자명", "Pass": "비밀번호", "BtnLogin": "로그인",
		"NoAccount": "계정이 없으신가요?", "SignUp": "가입하기", "RegTitle": "새 계정 만들기", "BtnReg": "등록",
		"ForceTitle": "비밀번호 변경 필요", "ForceMsg": "관리자가 비밀번호를 초기화했습니다. 새 비밀번호를 설정하십시오.",
		"Dashboard": "대시보드", "Profile": "설정", "Admin": "관리", "Logout": "로그아웃",
		"Total": "총 자산", "Expiring": "만료 예정 (30일)", "Urgent": "긴급 (7일)",
		"MyAssets": "자산 목록", "Export": "내보내기", "Import": "가져오기", "Backup": "DB 백업 다운로드",
		"Search": "분류, 이름, 날짜 검색...",
		"Cat": "카테고리", "Name": "이름", "Date": "만료일", "Action": "작업",
		"Add": "추가", "Details": "상세 정보 (계정/비번/메모)", "View": "보기", "Edit": "편집", "Del": "삭제",
		"Save": "저장", "Cancel": "취소", "Close": "닫기", "Copy": "복사", "Upload": "업로드",
		"NotifySettings": "알림 설정 (로그인 2FA 활성화)", "ChatID": "Telegram ID", "Email": "이메일",
		"Timezone": "시간대", "NotifyTime": "알림 시간", "GetID": "ID 가져오기", "TestBtn": "테스트 메시지 전송",
		"BotStatus": "봇 작동 중", "OpenBot": "봇 열기",
		"Security": "보안", "NewPass": "새 비밀번호", "ChangePass": "변경",
		"Msg_2FA": "2단계 인증 필요", "Msg_CodeSent": "코드 발송됨", "Code": "인증 코드", "Verify": "확인",
		"SelectMethod": "인증 방법 선택", "ViaTG": "Telegram으로 전송", "ViaEmail": "이메일로 전송",
		"GlobalSet": "전역 설정", "UserMgmt": "사용자 관리",
		"Role": "권한", "ResetPwd": "PW 초기화", "Reset2FA": "2FA 초기화", "DelUser": "삭제",
		"Ph_BotUser": "예: MyBot (@제외)", "Ph_BotToken": "예: 123456:ABC...",
		"Ph_Host": "예: smtp.gmail.com", "Ph_Port": "예: 465", "Ph_User": "이메일", "Ph_Pass": "비밀번호",
		"Ph_Cat": "카테고리 입력 또는 선택",
		"Msg_Add": "추가됨", "Msg_Upd": "업데이트됨", "Msg_Del": "삭제됨", 
		"Msg_Saved": "저장됨", "Msg_Sent": "전송 성공", "Msg_Fail": "전송 실패",
		"Msg_Reset": "PW가 123456으로 초기화됨", "Msg_2FACleared": "2FA 해제됨",
		"Msg_UserDel": "사용자 삭제됨", "Msg_PwdChanged": "비밀번호 변경됨",
		"Msg_RegSuccess": "가입 완료, 로그인해주세요", "Msg_LoginSuccess": "환영합니다", "Msg_Imported": "가져오기 성공",
		"Msg_LoginErr": "로그인 실패: 자격 증명이 잘못되었습니다", "Msg_CodeErr": "코드가 잘못되었거나 만료되었습니다",
		"ConfirmDel": "삭제하시겠습니까?", "ConfirmReset": "초기화하시겠습니까?",
		"Head_ID": "ID", "Head_User": "사용자", "Head_Role": "권한", "Head_Info": "자산/연동", "Head_IP": "최근 IP", "Head_Action": "작업",
		"TZ_Shanghai": "🇨🇳 중국 (북경)", "TZ_Tokyo": "🇯🇵 일본 (도쿄)", "TZ_Seoul": "🇰🇷 한국 (서울)",
		"TZ_Singapore": "🇸🇬 싱가포르", "TZ_London": "🇬🇧 영국 (런던)", "TZ_Berlin": "🇩🇪 독일 (베를린)",
		"TZ_NY": "🇺🇸 미국 (뉴욕)", "TZ_LA": "🇺🇸 미국 (LA)", "TZ_UTC": "🌐 UTC",
		"Notify_Title": "만료 알림", "Notify_Test": "테스트 알림", "Notify_Login": "로그인 인증",
		"Notify_Body_Login": "인증 코드:", "Notify_Body_IP": "IP:",
		"Notify_Body_Test": "ExpiryGuard 테스트 메시지입니다.",
		"Simulate_Title": "시뮬레이션 테스트", "Simulate_Desc": "알림 스타일을 미리 보기 위해 가짜 알림을 전송합니다.", "Simulate_Btn": "시뮬레이션 전송",
		"Sim_Domain": "도메인", "Sim_Server": "서버", "Sim_DaysLeft": "일 남음", "Sim_Expired": "만료됨!",
		"Import_Title": "일괄 가져오기 (CSV)", "Import_Desc": "CSV 파일을 업로드하십시오. 형식: 분류,이름,날짜(YYYY-MM-DD),상세",
	},
}

// ================= 模型定义 =================
type User struct {
	gorm.Model
	Username      string `gorm:"uniqueIndex"`
	Password      string
	Role          string
	ChatID        string
	Email         string
	NotifyTime    int
	Timezone      string
	Language      string
	LoginCode     string
	LoginCodeTime int64
	ForceReset    bool
	LastIP        string
	Items         []Item
}

type Item struct {
	gorm.Model
	UserID   uint
	Category string
	Name     string
	Date     string
	Detail   string
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

type PageData struct {
	Page         string
	User         User
	Items        []Item
	Users        []User
	Settings     map[string]string
	Stats        DashboardStats
	Message      string
	T            map[string]string
	Timezones    []TzOption
	Hours        []int
	LoginStep    string
	TwoFAMethods []string
	UserCats     []string
}

type DashboardStats struct {
	Total, Expiring, Urgent int
}
type TzOption struct {
	Val, LabelKey string
}

var CommonTimezones = []TzOption{
	{"Asia/Shanghai", "TZ_Shanghai"}, {"Asia/Tokyo", "TZ_Tokyo"},
	{"Asia/Seoul", "TZ_Seoul"}, {"Asia/Singapore", "TZ_Singapore"},
	{"Europe/London", "TZ_London"}, {"Europe/Berlin", "TZ_Berlin"},
	{"America/New_York", "TZ_NY"}, {"America/Los_Angeles", "TZ_LA"},
	{"UTC", "TZ_UTC"},
}

// ================= 全局 =================
var db *gorm.DB
var tmpl *template.Template

// ================= UI 模板 =================
const htmlContent = `
<!DOCTYPE html>
<html lang="{{.User.Language}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ExpiryGuard</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <script>tailwind.config={darkMode:'class',theme:{extend:{colors:{darkbg:'#0f172a',darkcard:'#1e293b'}}}}</script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
        body { font-family: 'Inter', sans-serif; transition: background-color 0.3s; }
        .toast { transform: translateX(100%); transition: all 0.4s cubic-bezier(0.68, -0.55, 0.27, 1.55); opacity: 0; }
        .toast.show { transform: translateX(0); opacity: 1; }
        .glass-card { background: rgba(255, 255, 255, 0.95); backdrop-filter: blur(10px); }
        .dark .glass-card { background: rgba(30, 41, 59, 0.9); }
        @keyframes breathe { 0%, 100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.7); } 50% { transform: scale(1.05); box-shadow: 0 0 20px 10px rgba(59, 130, 246, 0); } }
        .logo-breathe { animation: breathe 3s infinite; }
        /* 进度条 */
        .progress-bar { height: 4px; border-radius: 2px; background: #e2e8f0; margin-top: 4px; overflow: hidden; }
        .progress-fill { height: 100%; transition: width 0.5s ease; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 dark:bg-darkbg dark:text-slate-200 min-h-screen flex flex-col">

    <div id="toast-container" class="fixed top-5 right-5 z-50">
        {{if .Message}}
        <div id="toast" class="toast show flex items-center p-4 mb-4 bg-white/90 backdrop-blur rounded-xl shadow-2xl dark:bg-slate-800/90 border-l-4 border-blue-500">
            <div class="text-blue-500 mr-3"><i class="fas fa-info-circle text-xl"></i></div>
            <div class="text-sm font-medium">{{index .T .Message}}</div>
        </div>
        <script>setTimeout(() => { const t = document.getElementById('toast'); t.classList.remove('show'); setTimeout(()=>t.remove(), 400); }, 3500);</script>
        {{end}}
    </div>

    {{if or (eq .Page "login") (eq .Page "register") (eq .Page "2fa") (eq .Page "force_reset")}}
    <div class="flex-1 flex flex-col items-center justify-center bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 relative">
        <div class="absolute top-6 right-6 flex gap-3">
             <form id="langForm" action="/set-lang" method="POST">
                <input type="hidden" name="page" value="{{.Page}}">
                <select name="lang" onchange="this.form.submit()" class="bg-white/10 text-white text-xs p-2 rounded-lg backdrop-blur border border-white/20 outline-none cursor-pointer hover:bg-white/20 transition">
                    <option value="zh" {{if eq .User.Language "zh"}}selected{{end}}>🇨🇳 中文</option>
                    <option value="en" {{if eq .User.Language "en"}}selected{{end}}>🇺🇸 English</option>
                    <option value="ja" {{if eq .User.Language "ja"}}selected{{end}}>🇯🇵 日本語</option>
                    <option value="ko" {{if eq .User.Language "ko"}}selected{{end}}>🇰🇷 한국어</option>
                </select>
             </form>
             <button onclick="toggleDark()" class="bg-white/10 text-white p-2 rounded-lg w-9 h-9 flex items-center justify-center backdrop-blur border border-white/20 hover:bg-white/20 transition"><i class="fas fa-moon"></i></button>
        </div>

        <div class="glass-card p-10 rounded-3xl shadow-2xl w-full max-w-sm relative overflow-hidden transition-all duration-300">
            <div class="text-center mb-8">
                <div class="inline-flex items-center justify-center w-14 h-14 rounded-full bg-blue-100 text-blue-600 mb-4 logo-breathe"><i class="fas fa-shield-alt text-2xl"></i></div>
                <h1 class="text-2xl font-extrabold tracking-tight text-slate-900 dark:text-white">ExpiryGuard</h1>
            </div>

            {{if eq .Page "login"}}
                {{if eq .LoginStep "login"}}
                <form action="/login" method="POST" class="space-y-5">
                    <div class="relative"><i class="fas fa-user absolute left-4 top-3.5 text-slate-400"></i><input type="text" name="username" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none focus:ring-2 focus:ring-blue-500 transition" placeholder="{{.T.User}}" required></div>
                    <div class="relative"><i class="fas fa-lock absolute left-4 top-3.5 text-slate-400"></i><input type="password" name="password" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none focus:ring-2 focus:ring-blue-500 transition" placeholder="{{.T.Pass}}" required></div>
                    <button type="submit" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-3.5 rounded-xl shadow-lg shadow-blue-500/30 transition transform hover:-translate-y-0.5">{{.T.BtnLogin}}</button>
                </form>
                <div class="mt-8 text-center text-sm"><span class="text-slate-500">{{.T.NoAccount}}</span><a href="/register" class="text-blue-600 font-bold hover:underline ml-1">{{.T.SignUp}}</a></div>
                {{else if eq .LoginStep "2fa_select"}}
                <div class="space-y-4"><p class="text-center text-slate-500 text-sm mb-4">{{.T.Msg_2FA}}</p>{{range .TwoFAMethods}}{{if eq . "tg"}}<form action="/login/send-code" method="POST"><input type="hidden" name="method" value="tg"><button class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 rounded-xl flex items-center justify-center gap-2 shadow-lg transition"><i class="fab fa-telegram"></i> {{$.T.ViaTG}}</button></form>{{else if eq . "email"}}<form action="/login/send-code" method="POST"><input type="hidden" name="method" value="email"><button class="w-full bg-slate-600 hover:bg-slate-700 text-white font-bold py-3 rounded-xl flex items-center justify-center gap-2 shadow-lg transition"><i class="fas fa-envelope"></i> {{$.T.ViaEmail}}</button></form>{{end}}{{end}}</div>
                {{else if eq .LoginStep "2fa_input"}}
                <form action="/login/verify" method="POST" class="space-y-6"><p class="text-center text-green-500 text-sm font-medium"><i class="fas fa-check-circle"></i> {{.T.Msg_CodeSent}}</p><input type="text" name="code" class="w-full px-4 py-3 border rounded-xl text-center tracking-[0.5em] text-2xl font-bold font-mono dark:bg-slate-700 dark:border-slate-600 focus:ring-2 focus:ring-green-500 outline-none" placeholder="000000" required autofocus><button type="submit" class="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3 rounded-xl shadow-lg transition">{{.T.Verify}}</button></form>
                {{end}}
            {{else if eq .Page "register"}}
            <h2 class="text-center font-bold text-lg mb-6 text-slate-700 dark:text-slate-300">{{.T.RegTitle}}</h2>
            <form action="/register" method="POST" class="space-y-5"><div class="relative"><i class="fas fa-user absolute left-4 top-3.5 text-slate-400"></i><input type="text" name="username" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600" placeholder="{{.T.User}}" required></div><div class="relative"><i class="fas fa-lock absolute left-4 top-3.5 text-slate-400"></i><input type="password" name="password" class="w-full pl-10 pr-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600" placeholder="{{.T.Pass}}" required></div><button type="submit" class="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3.5 rounded-xl shadow-lg shadow-green-500/30 transition transform hover:-translate-y-0.5">{{.T.BtnReg}}</button></form>
            <div class="mt-6 text-center text-sm"><a href="/login" class="text-slate-500 hover:text-slate-800 dark:hover:text-white transition">← {{.T.LoginTitle}}</a></div>
            {{else if eq .Page "force_reset"}}
            <h2 class="text-center font-bold text-lg mb-2 text-red-500">{{.T.ForceTitle}}</h2><p class="text-center text-sm text-slate-500 mb-6">{{.T.ForceMsg}}</p><form action="/force-change-pwd" method="POST" class="space-y-4"><input type="password" name="password" class="w-full px-4 py-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600" placeholder="{{.T.NewPass}}" required><button type="submit" class="w-full bg-red-600 hover:bg-red-700 text-white font-bold py-3 rounded-xl shadow-lg">{{.T.ChangePass}}</button></form>
            {{end}}

             <div class="mt-10 text-center">
                <a href="https://t.me/TerrySiu98" target="_blank" class="text-xs text-gray-400 hover:text-blue-400 transition flex items-center justify-center gap-1 opacity-70 hover:opacity-100">
                    <span>Designed by Terry</span> <i class="fas fa-external-link-alt text-[9px]"></i>
                </a>
            </div>
        </div>
    </div>
    
    {{else}}
    <div class="flex flex-col md:flex-row min-h-screen">
        <aside class="w-64 bg-slate-900 text-slate-300 md:fixed md:inset-y-0 z-20 hidden md:flex flex-col shadow-2xl">
            <div class="p-6 flex items-center gap-3 text-white font-bold text-xl border-b border-slate-800"><i class="fas fa-shield-alt text-blue-500"></i> ExpiryGuard</div>
            <nav class="flex-1 p-4 space-y-1">
                <a href="/" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "home"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-th-large w-5 text-center"></i> <span>{{.T.Dashboard}}</span></a>
                <a href="/profile" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "profile"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-user-cog w-5 text-center"></i> <span>{{.T.Profile}}</span></a>
                {{if eq .User.Role "admin"}}
                <div class="text-[10px] font-bold text-slate-500 uppercase px-3 mb-2 mt-6">Admin</div>
                <a href="/admin" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "admin"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-cogs w-5 text-center"></i> <span>{{.T.GlobalSet}}</span></a>
                <a href="/admin/users" class="flex items-center space-x-3 px-3 py-3 rounded-lg {{if eq .Page "users"}}bg-blue-600 text-white shadow-lg{{else}}hover:bg-slate-800{{end}} transition"><i class="fas fa-users w-5 text-center"></i> <span>{{.T.UserMgmt}}</span></a>
                {{end}}
            </nav>
            <div class="p-4 border-t border-slate-800 flex justify-between items-center gap-2">
                <form id="sideLangForm" action="/set-lang" method="POST" class="flex-1">
                    <input type="hidden" name="page" value="{{.Page}}">
                    <select name="lang" onchange="this.form.submit()" class="w-full bg-slate-800 text-xs text-slate-300 p-1 rounded border border-slate-700 outline-none">
                        <option value="zh" {{if eq .User.Language "zh"}}selected{{end}}>🇨🇳 中文</option>
                        <option value="en" {{if eq .User.Language "en"}}selected{{end}}>🇺🇸 EN</option>
                        <option value="ja" {{if eq .User.Language "ja"}}selected{{end}}>🇯🇵 JA</option>
                        <option value="ko" {{if eq .User.Language "ko"}}selected{{end}}>🇰🇷 KO</option>
                    </select>
                </form>
                <button onclick="toggleDark()" class="text-slate-400 hover:text-white p-1"><i class="fas fa-moon"></i></button>
                <a href="/logout" class="text-red-400 hover:text-white p-1"><i class="fas fa-sign-out-alt"></i></a>
            </div>
        </aside>

        <main class="flex-1 md:ml-64 w-full flex flex-col">
             <div class="md:hidden bg-slate-900 text-white p-4 flex justify-between items-center sticky top-0 z-30 shadow-md">
                <span class="font-bold">ExpiryGuard</span>
                <div class="flex gap-4">
                    <button onclick="toggleMenu()" class="text-xl">☰</button>
                </div>
            </div>
            <div id="mobileMenu" class="fixed inset-0 bg-slate-900/95 z-40 hidden flex-col p-8 space-y-6 text-white text-lg font-bold md:hidden">
                <button onclick="toggleMenu()" class="absolute top-4 right-4 text-2xl">✕</button>
                <a href="/" class="block border-b border-white/10 pb-2">{{.T.Dashboard}}</a>
                <a href="/profile" class="block border-b border-white/10 pb-2">{{.T.Profile}}</a>
                {{if eq .User.Role "admin"}}
                <div class="text-xs text-slate-500 uppercase mt-4">Admin</div>
                <a href="/admin" class="block border-b border-white/10 pb-2">{{.T.GlobalSet}}</a>
                <a href="/admin/users" class="block border-b border-white/10 pb-2">{{.T.UserMgmt}}</a>
                {{end}}
                <a href="/logout" class="block text-red-400 pt-4">{{.T.Logout}}</a>
            </div>

            <div class="max-w-7xl mx-auto p-4 md:p-8 w-full flex-1">
                {{if eq .Page "home"}}
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                    <div class="bg-gradient-to-r from-blue-500 to-blue-600 text-white p-5 rounded-xl shadow-lg flex justify-between items-center transform hover:-translate-y-1 transition"><div><div class="text-blue-100 text-xs font-bold uppercase">{{.T.Total}}</div><div class="text-3xl font-bold">{{.Stats.Total}}</div></div><div class="text-blue-300 text-4xl opacity-50"><i class="fas fa-cube"></i></div></div>
                    <div class="bg-gradient-to-r from-orange-400 to-orange-500 text-white p-5 rounded-xl shadow-lg flex justify-between items-center transform hover:-translate-y-1 transition"><div><div class="text-orange-100 text-xs font-bold uppercase">{{.T.Expiring}}</div><div class="text-3xl font-bold">{{.Stats.Expiring}}</div></div><div class="text-orange-200 text-4xl opacity-50"><i class="fas fa-clock"></i></div></div>
                    <div class="bg-gradient-to-r from-red-500 to-red-600 text-white p-5 rounded-xl shadow-lg flex justify-between items-center transform hover:-translate-y-1 transition"><div><div class="text-red-100 text-xs font-bold uppercase">{{.T.Urgent}}</div><div class="text-3xl font-bold">{{.Stats.Urgent}}</div></div><div class="text-red-300 text-4xl opacity-50"><i class="fas fa-exclamation-triangle"></i></div></div>
                </div>

                <div class="bg-white dark:bg-darkcard rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 p-5">
                    <div class="flex flex-col md:flex-row justify-between items-center mb-4 gap-3">
                        <div class="flex gap-2 items-center w-full md:w-auto"><h2 class="text-lg font-bold">{{.T.MyAssets}}</h2><a href="/export" class="text-xs bg-slate-100 dark:bg-slate-700 px-3 py-1.5 rounded-lg hover:bg-slate-200 transition font-medium text-slate-600 dark:text-slate-300"><i class="fas fa-download mr-1"></i>{{.T.Export}}</a><button onclick="openModal('importModal')" class="bg-slate-100 dark:bg-slate-700 px-3 py-1.5 text-xs font-bold rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition text-slate-600 dark:text-slate-300"><i class="fas fa-file-import mr-1"></i> {{.T.Import}}</button></div>
                        <div class="relative w-full md:w-64"><input type="text" id="searchInput" placeholder="{{.T.Search}}" class="w-full pl-9 pr-4 py-2 bg-slate-50 dark:bg-slate-800 border-none rounded-lg text-sm focus:ring-2 focus:ring-blue-500 transition"><i class="fas fa-search absolute left-3 top-2.5 text-slate-400 text-xs"></i></div>
                    </div>
                    <button onclick="openModal('addModal')" class="w-full bg-slate-800 dark:bg-blue-600 text-white py-3 rounded-lg text-sm font-bold shadow-md hover:bg-slate-900 transition mb-4"><i class="fas fa-plus mr-1"></i> {{.T.Add}}</button>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left border-collapse min-w-[600px]">
                            <thead class="text-xs text-slate-400 uppercase border-b dark:border-slate-700"><tr><th class="pb-3 pl-2">{{.T.Cat}}</th><th class="pb-3">{{.T.Name}}</th><th class="pb-3 w-1/3">{{.T.Date}}</th><th class="pb-3 text-right pr-2">{{.T.Action}}</th></tr></thead>
                            <tbody class="text-sm">
                                {{range .Items}}
                                <tr class="border-b dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 search-item transition-all duration-200 hover:shadow-sm">
                                    <td class="py-3 pl-2"><span class="px-2.5 py-1 bg-slate-100 dark:bg-slate-700 rounded-full text-xs font-medium">{{.Category}}</span></td>
                                    <td class="py-3 font-semibold text-slate-700 dark:text-slate-200 search-text">{{.Name}}</td>
                                    <td class="py-3">
                                        <div class="font-mono text-blue-600 dark:text-blue-400 font-medium">{{.Date}}</div>
                                        <div class="progress-bar dark:bg-slate-700" data-date="{{.Date}}"><div class="progress-fill"></div></div>
                                    </td>
                                    <td class="py-3 text-right pr-2 space-x-2">
                                        <button onclick="openView('{{.Name}}', '{{.Detail}}')" class="text-green-600 hover:text-green-700 bg-green-50 dark:bg-green-900/30 px-2.5 py-1.5 rounded-lg text-xs font-medium transition"><i class="fas fa-eye"></i> {{$.T.View}}</button>
                                        <button onclick="openEdit('{{.ID}}','{{.Category}}','{{.Name}}','{{.Date}}','{{.Detail}}')" class="text-blue-500 hover:text-blue-600 p-1"><i class="fas fa-edit"></i></button>
                                        <form action="/item/del" method="POST" onsubmit="return confirm('{{$.T.ConfirmDel}}')" class="inline"><input type="hidden" name="id" value="{{.ID}}"><button class="text-slate-400 hover:text-red-500 p-1"><i class="fas fa-trash-alt"></i></button></form>
                                    </td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{else if eq .Page "profile"}}
                <div class="bg-white dark:bg-darkcard rounded-xl shadow-sm p-6 border border-slate-100 dark:border-slate-700">
                    <h2 class="text-xl font-bold mb-6 flex items-center gap-2"><i class="fas fa-bell text-blue-500"></i> {{.T.NotifySettings}}</h2>
                    {{if and .Settings.tg_bot_username .Settings.tg_token}}
                    <div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-xl mb-6 flex justify-between items-center border border-blue-100 dark:border-blue-800"><div class="text-blue-800 dark:text-blue-300 font-bold flex items-center gap-2"><div class="relative flex h-3 w-3"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span><span class="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span></div> {{.T.BotStatus}}</div><a href="https://t.me/{{.Settings.tg_bot_username}}" target="_blank" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-bold shadow-md transition">{{.T.OpenBot}}</a></div>
                    {{end}}
                    <form action="/profile/update" method="POST" class="space-y-5">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5"><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.ChatID}}</label><div class="flex gap-2"><input type="text" name="chat_id" value="{{.User.ChatID}}" class="flex-1 p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 focus:ring-2 focus:ring-blue-500 outline-none"><button type="button" onclick="window.open('https://t.me/userinfobot')" class="px-3 bg-slate-100 dark:bg-slate-700 text-xs font-bold rounded-lg hover:bg-slate-200 transition">{{.T.GetID}}</button></div></div><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.Email}}</label><input type="email" name="email" value="{{.User.Email}}" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600 focus:ring-2 focus:ring-blue-500 outline-none"></div></div>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5"><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.Timezone}}</label><select name="timezone" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600">{{range .Timezones}}<option value="{{.Val}}" {{if eq $.User.Timezone .Val}}selected{{end}}>{{index $.T .LabelKey}}</option>{{end}}</select></div><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">{{.T.NotifyTime}}</label><select name="notify_time" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600">{{range .Hours}}<option value="{{.}}" {{if eq $.User.NotifyTime .}}selected{{end}}>{{printf "%02d:00" .}}</option>{{end}}</select></div></div>
                        <button type="submit" class="w-full bg-slate-800 dark:bg-blue-600 text-white py-3 rounded-lg font-bold shadow-lg hover:bg-slate-900 transition">{{.T.Save}}</button>
                    </form>
                    <form action="/test-notify" method="POST" class="mt-6 pt-6 border-t dark:border-slate-700"><button class="w-full bg-purple-50 text-purple-600 py-3 rounded-lg border border-purple-200 text-sm font-bold dark:bg-purple-900/10 dark:border-purple-800 dark:text-purple-300 hover:bg-purple-100 transition"><i class="fas fa-paper-plane mr-1"></i> {{.T.TestBtn}}</button></form>
                </div>
                {{else if eq .Page "admin"}}
                <div class="max-w-4xl mx-auto bg-white dark:bg-darkcard rounded-xl shadow-sm border border-slate-100 dark:border-slate-700 p-6">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-xl font-bold flex items-center gap-2"><i class="fas fa-cogs text-slate-500"></i> {{.T.GlobalSet}}</h2>
                        <a href="/admin/backup" target="_blank" class="bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 px-3 py-1.5 rounded-lg text-xs font-bold hover:bg-slate-200 dark:hover:bg-slate-600 transition"><i class="fas fa-database mr-1"></i> {{.T.Backup}}</a>
                    </div>
                    <form action="/admin/update" method="POST" class="space-y-5">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5"><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">Bot Username</label><input type="text" name="tg_bot_username" value="{{.Settings.tg_bot_username}}" placeholder="{{.T.Ph_BotUser}}" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"></div><div><label class="text-xs font-bold text-slate-400 uppercase mb-1 block">Bot Token</label><input type="password" name="tg_token" value="{{.Settings.tg_token}}" placeholder="{{.T.Ph_BotToken}}" class="w-full p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"></div></div>
                        <hr class="dark:border-slate-700">
                        <div class="grid grid-cols-2 gap-5"><input type="text" name="smtp_host" value="{{.Settings.smtp_host}}" placeholder="{{.T.Ph_Host}}" class="p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"><input type="text" name="smtp_port" value="{{.Settings.smtp_port}}" placeholder="{{.T.Ph_Port}}" class="p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"><input type="text" name="smtp_user" value="{{.Settings.smtp_user}}" placeholder="{{.T.Ph_User}}" class="p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"><input type="password" name="smtp_pass" value="{{.Settings.smtp_pass}}" placeholder="{{.T.Ph_Pass}}" class="p-2.5 border rounded-lg dark:bg-slate-800 dark:border-slate-600"></div>
                        <button type="submit" class="w-full bg-slate-800 text-white py-3 rounded-lg font-bold dark:bg-blue-600 shadow-lg">{{.T.Save}}</button>
                    </form>
                    
                    <div class="mt-8 pt-8 border-t dark:border-slate-700">
                        <h3 class="text-lg font-bold mb-3 flex items-center gap-2 text-slate-700 dark:text-slate-300"><i class="fas fa-vial"></i> {{.T.Simulate_Title}}</h3>
                        <p class="text-sm text-slate-500 mb-4">{{.T.Simulate_Desc}}</p>
                        <form action="/admin/simulate" method="POST">
                            <button type="submit" class="bg-orange-500 hover:bg-orange-600 text-white px-5 py-2.5 rounded-lg font-bold shadow transition"><i class="fas fa-bell mr-1"></i> {{.T.Simulate_Btn}}</button>
                        </form>
                    </div>
                </div>
                {{else if eq .Page "users"}}
                <div class="bg-white dark:bg-darkcard rounded-xl p-6">
                    <h2 class="text-xl font-bold mb-4">{{.T.UserMgmt}}</h2>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left border-collapse min-w-[600px]"><thead class="text-xs uppercase text-slate-400 border-b dark:border-slate-700"><tr><th class="pb-2">{{.T.Head_ID}}</th><th class="pb-2">{{.T.Head_User}}</th><th class="pb-2">{{.T.Head_Role}}</th><th class="pb-2">{{.T.Head_Info}}</th><th class="pb-2">{{.T.Head_IP}}</th><th class="pb-2">{{.T.Head_Action}}</th></tr></thead>
                        <tbody>{{range .Users}}<tr class="border-b dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800"><td class="py-3 text-xs text-slate-400">#{{.ID}}</td><td class="font-bold">{{.Username}}</td><td><span class="px-2 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-xs">{{.Role}}</span></td><td class="text-xs"><span class="mr-2 font-bold">{{len .Items}} Assets</span>
                        {{if .ChatID}}<span class="text-blue-500 mr-1" title="TG Bound">🔵</span>{{else}}<span class="text-red-400 mr-1" title="Unbound">🔴</span>{{end}}
                        {{if .Email}}<span class="text-blue-500" title="Email Bound">🔵</span>{{else}}<span class="text-red-400" title="Unbound">🔴</span>{{end}}
                        </td><td class="text-xs font-mono text-slate-400">{{.LastIP}}</td><td class="space-x-1">{{if ne .Role "admin"}}<form action="/admin/reset-pwd" method="POST" onsubmit="return confirm('{{$.T.ConfirmReset}}')" class="inline"><input type="hidden" name="id" value="{{.ID}}"><button class="text-blue-500 text-xs font-bold bg-blue-50 dark:bg-blue-900/30 px-2 py-1 rounded hover:bg-blue-100">{{$.T.ResetPwd}}</button></form><form action="/admin/reset-2fa" method="POST" onsubmit="return confirm('{{$.T.ConfirmReset}}')" class="inline"><input type="hidden" name="id" value="{{.ID}}"><button class="text-orange-500 text-xs font-bold bg-orange-50 dark:bg-orange-900/30 px-2 py-1 rounded hover:bg-orange-100">{{$.T.Reset2FA}}</button></form><form action="/admin/users/del" method="POST" onsubmit="return confirm('{{$.T.ConfirmDel}}')" class="inline"><input type="hidden" name="id" value="{{.ID}}"><button class="text-red-500 text-xs font-bold bg-red-50 dark:bg-red-900/30 px-2 py-1 rounded hover:bg-red-100">{{$.T.DelUser}}</button></form>{{end}}</td></tr>{{end}}</tbody></table>
                    </div>
                </div>
                {{end}}
            </div>
            
            <footer class="text-center p-6 mt-auto pb-20 md:pb-6"><a href="https://t.me/TerrySiu98" target="_blank" class="text-slate-400 text-xs hover:text-blue-500 transition font-medium flex items-center justify-center gap-1"><i class="fab fa-telegram"></i> Designed by Terry</a></footer>
        </main>
    </div>

    <div id="addModal" class="fixed inset-0 hidden items-center justify-center z-50 bg-black/50 backdrop-blur-sm"><div class="glass-card p-6 rounded-2xl w-full max-w-lg shadow-2xl"><h3 class="font-bold mb-4 text-lg">{{.T.Add}}</h3><form action="/item/add" method="POST" class="space-y-4">
        <input type="text" name="category" list="cat_list" placeholder="{{.T.Ph_Cat}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required>
        <datalist id="cat_list">{{range .UserCats}}<option value="{{.}}">{{end}}</datalist>
        <input type="text" name="name" placeholder="{{.T.Name}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required><input type="date" name="date" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required><textarea name="detail" rows="6" placeholder="{{.T.Details}}" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 font-mono text-sm outline-none"></textarea><div class="flex gap-3"><button type="button" onclick="closeModal('addModal')" class="flex-1 py-3 bg-slate-100 text-slate-600 rounded-xl hover:bg-slate-200 transition font-bold">{{.T.Cancel}}</button><button type="submit" class="flex-1 py-3 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition font-bold shadow-lg">{{.T.Save}}</button></div></form></div></div>
    
    <div id="importModal" class="fixed inset-0 hidden items-center justify-center z-50 bg-black/50 backdrop-blur-sm"><div class="glass-card p-6 rounded-2xl w-full max-w-lg shadow-2xl"><h3 class="font-bold mb-2 text-lg">{{.T.Import_Title}}</h3><p class="text-sm text-slate-500 mb-4">{{.T.Import_Desc}}</p><form action="/item/import" method="POST" enctype="multipart/form-data" class="space-y-4"><input type="file" name="file" accept=".csv" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600"><div class="flex gap-3"><button type="button" onclick="closeModal('importModal')" class="flex-1 py-3 bg-slate-100 text-slate-600 rounded-xl hover:bg-slate-200 transition font-bold">{{.T.Cancel}}</button><button type="submit" class="flex-1 py-3 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition font-bold shadow-lg">{{.T.Upload}}</button></div></form></div></div>

    <div id="editModal" class="fixed inset-0 hidden items-center justify-center z-50 bg-black/50 backdrop-blur-sm"><div class="glass-card p-6 rounded-2xl w-full max-w-lg shadow-2xl"><h3 class="font-bold mb-4 text-lg">{{.T.Edit}}</h3><form action="/item/update" method="POST" class="space-y-4"><input type="hidden" name="id" id="edit_id"><input type="text" name="category" id="edit_category" list="cat_list" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none" required><input type="text" name="name" id="edit_name" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none"><input type="date" name="date" id="edit_date" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 outline-none"><textarea name="detail" id="edit_detail" rows="6" class="w-full p-3 border rounded-xl dark:bg-slate-700 dark:border-slate-600 font-mono text-sm outline-none"></textarea><div class="flex gap-3"><button type="button" onclick="closeModal('editModal')" class="flex-1 py-3 bg-slate-100 text-slate-600 rounded-xl hover:bg-slate-200 transition font-bold">{{.T.Cancel}}</button><button type="submit" class="flex-1 py-3 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition font-bold shadow-lg">{{.T.Save}}</button></div></form></div></div>
    <div id="viewModal" class="fixed inset-0 hidden items-center justify-center z-50 bg-black/50 backdrop-blur-sm"><div class="glass-card p-6 rounded-2xl w-full max-w-2xl mx-4 shadow-2xl"><div class="flex justify-between items-start mb-4"><h3 class="font-bold text-xl" id="view_title"></h3><button onclick="copyContent()" class="text-blue-500 hover:text-blue-600 text-sm font-bold flex items-center gap-1 transition"><i class="fas fa-copy"></i> {{.T.Copy}}</button></div><div class="bg-slate-50 dark:bg-slate-900 p-6 rounded-xl border dark:border-slate-700 mb-6 max-h-[60vh] overflow-y-auto relative"><pre id="view_content" class="whitespace-pre-wrap font-mono text-sm break-all text-slate-700 dark:text-slate-300"></pre></div><button onclick="closeModal('viewModal')" class="w-full py-3 bg-slate-800 text-white rounded-xl font-bold hover:bg-slate-900 transition shadow-lg">{{.T.Close}}</button></div></div>
    {{end}}

    <script>
        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const term = e.target.value.toLowerCase();
                document.querySelectorAll('.search-item').forEach(item => {
                    const text = item.innerText.toLowerCase();
                    item.style.display = text.includes(term) ? '' : 'none';
                });
            });
        }
        function toggleMenu(){document.getElementById('mobileMenu').classList.toggle('hidden');document.getElementById('mobileMenu').classList.toggle('flex');}
        function toggleDark(){document.documentElement.classList.toggle('dark');localStorage.setItem('theme',document.documentElement.classList.contains('dark')?'dark':'light');}
        if(localStorage.getItem('theme')==='dark'){document.documentElement.classList.add('dark');}
        function openModal(id){document.getElementById(id).classList.remove('hidden');document.getElementById(id).classList.add('flex');}
        function closeModal(id){document.getElementById(id).classList.add('hidden');document.getElementById(id).classList.remove('flex');}
        function openEdit(id,cat,name,date,detail){document.getElementById('edit_id').value=id;document.getElementById('edit_category').value=cat;document.getElementById('edit_name').value=name;document.getElementById('edit_date').value=date;document.getElementById('edit_detail').value=detail;openModal('editModal');}
        function openView(name,detail){document.getElementById('view_title').innerText=name;document.getElementById('view_content').innerText=detail;openModal('viewModal');}
        function copyContent(){const text=document.getElementById('view_content').innerText;navigator.clipboard.writeText(text).then(()=>{const btn=document.querySelector('#viewModal button i.fa-copy').parentNode;const original=btn.innerHTML;btn.innerHTML='<i class="fas fa-check"></i> Copied!';setTimeout(()=>btn.innerHTML=original,2000);}).catch(err=>{console.error('Failed to copy',err);});}
        // 进度条渲染
        document.querySelectorAll('.progress-bar').forEach(bar => {
            const dateStr = bar.dataset.date;
            const targetDate = new Date(dateStr);
            const now = new Date();
            const diffTime = targetDate - now;
            const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
            let color = '#22c55e'; // Green
            if (diffDays <= 30) color = '#f97316'; // Orange
            if (diffDays <= 7) color = '#ef4444'; // Red
            let width = Math.min(100, Math.max(0, (diffDays / 365) * 100));
            if (diffDays < 0) { width = 100; color = '#94a3b8'; } 
            const fill = bar.querySelector('.progress-fill');
            fill.style.width = width + '%';
            fill.style.backgroundColor = color;
        });
    </script>
</body>
</html>
`

// ================= 逻辑处理 =================

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	db.AutoMigrate(&User{}, &Item{}, &Setting{}) 
	initData()
	tmpl = template.Must(template.New("main").Parse(htmlContent))

	http.HandleFunc("/", auth(homeHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/login/send-code", loginSendCodeHandler)
	http.HandleFunc("/login/verify", loginVerifyHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/set-lang", setLangHandler)
	http.HandleFunc("/force-change-pwd", forceChangePwdHandler)

	http.HandleFunc("/export", auth(exportHandler))
	http.HandleFunc("/item/import", auth(importHandler)) // ✅ 新增：导入逻辑
	http.HandleFunc("/item/add", auth(addItemHandler))
	http.HandleFunc("/item/del", auth(delItemHandler))
	http.HandleFunc("/item/update", auth(updateItemHandler))
	http.HandleFunc("/profile", auth(profileHandler))
	http.HandleFunc("/profile/update", auth(profileUpdateHandler))
	http.HandleFunc("/profile/password", auth(changePwdHandler))
	http.HandleFunc("/test-notify", auth(testNotifyHandler))
	
	http.HandleFunc("/admin", auth(adminHandler))
	http.HandleFunc("/admin/update", auth(adminUpdateHandler))
	http.HandleFunc("/admin/users", auth(adminUsersHandler))
	http.HandleFunc("/admin/users/del", auth(adminDelUserHandler))
	http.HandleFunc("/admin/reset-pwd", auth(adminResetPwdHandler))
	http.HandleFunc("/admin/reset-2fa", auth(adminReset2FAHandler))
	http.HandleFunc("/admin/simulate", auth(adminSimulateHandler))
	http.HandleFunc("/admin/backup", auth(adminBackupHandler))

	go startScheduler()
	fmt.Println("🚀 ExpiryGuard Ultimate Started :8080")
	http.ListenAndServe(":8080", nil)
}

// ----------------- Auth Logic -----------------
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" { renderPage(w, r, "login", nil, r.URL.Query().Get("msg")); return }
	u, p := r.FormValue("username"), r.FormValue("password")
	var user User
	if db.Where("username = ?", u).First(&user).Error != nil || user.Password != p { renderPage(w, r, "login", nil, "Msg_LoginErr"); return }
	
	user.LastIP = getClientIP(r); db.Save(&user)

	if user.ForceReset { 
		http.SetCookie(w, &http.Cookie{Name: "temp_uid", Value: fmt.Sprint(user.ID), Path: "/", MaxAge: 300})
		data := getPageData(user, "force_reset", "")
		tmpl.Execute(w, data)
		return 
	}

	methods := []string{}
	if user.ChatID != "" { methods = append(methods, "tg") }
	if user.Email != "" { methods = append(methods, "email") }
	
	if len(methods) == 0 { 
		setAuthCookie(w, user.ID)
		redirectWithMsg(w, r, "/", "Msg_LoginSuccess")
		return 
	}
	
	http.SetCookie(w, &http.Cookie{Name: "temp_uid", Value: fmt.Sprint(user.ID), Path: "/", MaxAge: 300})
	data := getPageData(user, "login", ""); data.LoginStep = "2fa_select"; data.TwoFAMethods = methods
	tmpl.Execute(w, data)
}

// ✅ 修改：去掉了 msg 开头的重复标题，只保留正文
func loginSendCodeHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("temp_uid"); if c == nil { http.Redirect(w, r, "/login", http.StatusFound); return }
	var user User; db.First(&user, c.Value)
	code := generateCode(); user.LoginCode = code; user.LoginCodeTime = time.Now().Add(5 * time.Minute).Unix(); db.Save(&user)
	method := r.FormValue("method"); settings := getSettingsMap()
	
	// 修改：去掉了开头的 Title 变量，避免邮件中重复
	msg := fmt.Sprintf("%s <b style='font-size:18px;color:#2563eb'>%s</b><br><br>%s %s<br><br><i>🛡️ ExpiryGuard System</i>", LangMap[user.Language]["Notify_Body_Login"], code, LangMap[user.Language]["Notify_Body_IP"], user.LastIP)
	
	if method == "tg" { 
		// TG 还是需要标题的
		tgMsg := fmt.Sprintf("<b>%s</b><br><br>%s", LangMap[user.Language]["Notify_Login"], msg)
		sendTg(settings["tg_token"], user.ChatID, tgMsg) 
	} else { 
		// 邮件 Subject 会自动充当标题
		sendEmail(settings, user.Email, LangMap[user.Language]["Notify_Login"], msg) 
	}
	
	data := getPageData(user, "login", ""); data.LoginStep = "2fa_input"; tmpl.Execute(w, data)
}

func loginVerifyHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("temp_uid"); if c == nil { http.Redirect(w, r, "/login", http.StatusFound); return }
	var user User; db.First(&user, c.Value)
	if r.FormValue("code") == user.LoginCode && time.Now().Unix() < user.LoginCodeTime { 
		user.LoginCode = ""; db.Save(&user)
		if user.ForceReset { data := getPageData(user, "force_reset", ""); tmpl.Execute(w, data); return }
		setAuthCookie(w, user.ID); redirectWithMsg(w, r, "/", "Msg_LoginSuccess")
	} else { renderPage(w, r, "login", nil, "Msg_CodeErr") }
}

func forceChangePwdHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("temp_uid"); if c == nil { http.Redirect(w, r, "/login", http.StatusFound); return }
	var user User; db.First(&user, c.Value)
	user.Password = r.FormValue("password"); user.ForceReset = false; db.Save(&user)
	http.SetCookie(w, &http.Cookie{Name: "temp_uid", MaxAge: -1})
	redirectWithMsg(w, r, "/login", "Msg_PwdChanged")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" { renderPage(w, r, "register", nil, ""); return }
	u, p := r.FormValue("username"), r.FormValue("password")
	var count int64; db.Model(&User{}).Where("username = ?", u).Count(&count)
	if count > 0 { renderPage(w, r, "register", nil, "Username exists"); return }
	role := "user"; db.Model(&User{}).Count(&count); if count == 0 { role = "admin" }
	lang := "zh"; if c, err := r.Cookie("lang"); err == nil { lang = c.Value }
	newUser := User{Username: u, Password: p, Role: role, Language: lang, Timezone: "Asia/Shanghai", NotifyTime: 9, LastIP: getClientIP(r)}
	db.Create(&newUser); redirectWithMsg(w, r, "/login", "Msg_RegSuccess")
}

func setLangHandler(w http.ResponseWriter, r *http.Request) {
	lang, page := r.FormValue("lang"), r.FormValue("page")
	http.SetCookie(w, &http.Cookie{Name: "lang", Value: lang, Path: "/", MaxAge: 86400 * 365})
	c, err := r.Cookie("uid"); if err == nil { var user User; if db.First(&user, c.Value).Error == nil { user.Language = lang; db.Save(&user) } }
	target := "/"; if page == "login" || page == "register" { target = "/login" }; http.Redirect(w, r, target, http.StatusFound)
}

// ----------------- Helper -----------------
func initData() {} 
func generateCode() string { n, _ := rand.Int(rand.Reader, big.NewInt(900000)); return fmt.Sprintf("%06d", n.Int64()+100000) }
func setAuthCookie(w http.ResponseWriter, uid uint) { http.SetCookie(w, &http.Cookie{Name: "uid", Value: fmt.Sprint(uid), Path: "/", HttpOnly: true, MaxAge: 86400 * 30}) }
func renderPage(w http.ResponseWriter, r *http.Request, page string, u *User, msg string) { lang := "zh"; if c, err := r.Cookie("lang"); err == nil { lang = c.Value }; dummyUser := User{Language: lang}; if u != nil { dummyUser = *u }; data := getPageData(dummyUser, page, msg); tmpl.Execute(w, data) }
func getPageData(u User, page string, msg string) PageData { lang := u.Language; if lang == "" { lang = "zh" }; hours := make([]int, 24); for i := 0; i < 24; i++ { hours[i] = i }; data := PageData{Page: page, User: u, Message: msg, T: LangMap[lang], Timezones: CommonTimezones, Hours: hours}; if page == "login" && data.LoginStep == "" { data.LoginStep = "login" }; return data }
func auth(next func(w http.ResponseWriter, r *http.Request, u User)) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) { c, err := r.Cookie("uid"); if err != nil { http.Redirect(w, r, "/login", http.StatusFound); return }; var user User; if db.First(&user, c.Value).Error != nil { http.Redirect(w, r, "/login", http.StatusFound); return }; if user.ForceReset { http.SetCookie(w, &http.Cookie{Name: "uid", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusFound); return }; next(w, r, user) } }

// ✅ 修改：优先取 CF-Connecting-IP，并对 X-Forwarded-For 进行切割取第一个
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" { return ip }
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		if strings.Contains(ip, ",") {
			parts := strings.Split(ip, ",")
			return strings.TrimSpace(parts[0])
		}
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" { return ip }
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func redirectWithMsg(w http.ResponseWriter, r *http.Request, path, msg string) { http.Redirect(w, r, path+"?msg="+url.QueryEscape(msg), http.StatusFound) }

// ----------------- App Handlers -----------------
func homeHandler(w http.ResponseWriter, r *http.Request, u User) { 
	var items []Item; db.Where("user_id = ?", u.ID).Find(&items); sort.Slice(items, func(i, j int) bool { return items[i].Date < items[j].Date })
	var userCats []string; db.Model(&Item{}).Where("user_id = ?", u.ID).Distinct("category").Pluck("category", &userCats)
	stats := DashboardStats{Total: len(items)}; now := time.Now()
	for _, it := range items { t, _ := time.Parse("2006-01-02", it.Date); days := int(t.Sub(now).Hours() / 24); if days <= 7 { stats.Urgent++ }; if days <= 30 { stats.Expiring++ } }
	data := getPageData(u, "home", r.URL.Query().Get("msg")); data.Items = items; data.UserCats = userCats; data.Stats = stats; tmpl.Execute(w, data) 
}
func addItemHandler(w http.ResponseWriter, r *http.Request, u User) { if r.Method == "POST" { db.Create(&Item{UserID: u.ID, Category: r.FormValue("category"), Name: r.FormValue("name"), Date: r.FormValue("date"), Detail: r.FormValue("detail")}); redirectWithMsg(w, r, "/", "Msg_Add") } }
func updateItemHandler(w http.ResponseWriter, r *http.Request, u User) { var item Item; if db.Where("id = ? AND user_id = ?", r.FormValue("id"), u.ID).First(&item).Error == nil { item.Category = r.FormValue("category"); item.Name = r.FormValue("name"); item.Date = r.FormValue("date"); item.Detail = r.FormValue("detail"); db.Save(&item); redirectWithMsg(w, r, "/", "Msg_Upd") } }
func delItemHandler(w http.ResponseWriter, r *http.Request, u User) { db.Where("id = ? AND user_id = ?", r.FormValue("id"), u.ID).Delete(&Item{}); redirectWithMsg(w, r, "/", "Msg_Del") }
func exportHandler(w http.ResponseWriter, r *http.Request, u User) { var items []Item; db.Where("user_id = ?", u.ID).Find(&items); b := &bytes.Buffer{}; wr := csv.NewWriter(b); wr.Write([]string{"Category", "Name", "Date", "Details"}); for _, i := range items { wr.Write([]string{i.Category, i.Name, i.Date, i.Detail}) }; wr.Flush(); w.Header().Set("Content-Type", "text/csv"); w.Header().Set("Content-Disposition", "attachment;filename=assets.csv"); w.Write(b.Bytes()) }
// ✅ 新增：导入 Handler
func importHandler(w http.ResponseWriter, r *http.Request, u User) {
	file, _, err := r.FormFile("file")
	if err != nil { redirectWithMsg(w, r, "/", "Msg_ImportFail"); return }
	defer file.Close()
	reader := csv.NewReader(file); _, _ = reader.Read() // Skip header
	for {
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { continue }
		if len(record) >= 3 {
			// 简单的日期验证，若失败则忽略
			if _, err := time.Parse("2006-01-02", record[2]); err == nil {
				detail := ""; if len(record) > 3 { detail = record[3] }
				db.Create(&Item{UserID: u.ID, Category: record[0], Name: record[1], Date: record[2], Detail: detail})
			}
		}
	}
	redirectWithMsg(w, r, "/", "Msg_Imported")
}
func profileHandler(w http.ResponseWriter, r *http.Request, u User) { data := getPageData(u, "profile", r.URL.Query().Get("msg")); data.Settings = getSettingsMap(); tmpl.Execute(w, data) }
func profileUpdateHandler(w http.ResponseWriter, r *http.Request, u User) { u.ChatID = r.FormValue("chat_id"); u.Email = r.FormValue("email"); u.Timezone = r.FormValue("timezone"); fmt.Sscan(r.FormValue("notify_time"), &u.NotifyTime); db.Save(&u); redirectWithMsg(w, r, "/profile", "Msg_Saved") }
func changePwdHandler(w http.ResponseWriter, r *http.Request, u User) { 
	u.Password = r.FormValue("password"); 
	if r.URL.Path == "/profile/password" { db.Save(&u); redirectWithMsg(w, r, "/profile", "Msg_PwdChanged")
	} else { u.ForceReset = false; db.Save(&u); http.SetCookie(w, &http.Cookie{Name: "temp_uid", MaxAge: -1}); redirectWithMsg(w, r, "/login", "Msg_PwdChanged") }
}
func testNotifyHandler(w http.ResponseWriter, r *http.Request, u User) { s := getSettingsMap(); sent := false; msg := fmt.Sprintf("<b>%s</b><br><br>%s<br><br><i>🛡️ ExpiryGuard System</i>", LangMap[u.Language]["Notify_Test"], LangMap[u.Language]["Notify_Body_Test"]); if s["tg_token"] != "" && u.ChatID != "" { sendTg(s["tg_token"], u.ChatID, msg); sent = true }; if s["smtp_host"] != "" && u.Email != "" { sendEmail(s, u.Email, LangMap[u.Language]["Notify_Test"], msg); sent = true }; if sent { redirectWithMsg(w, r, "/profile", "Msg_Sent") } else { redirectWithMsg(w, r, "/profile", "Msg_Fail") } }
func logoutHandler(w http.ResponseWriter, r *http.Request) { http.SetCookie(w, &http.Cookie{Name: "uid", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusFound) }
func adminHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { http.Redirect(w, r, "/", http.StatusFound); return }; data := getPageData(u, "admin", r.URL.Query().Get("msg")); data.Settings = getSettingsMap(); tmpl.Execute(w, data) }
func adminUpdateHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { return }; saveSetting("tg_bot_username", r.FormValue("tg_bot_username")); saveSetting("tg_token", r.FormValue("tg_token")); saveSetting("smtp_host", r.FormValue("smtp_host")); saveSetting("smtp_port", r.FormValue("smtp_port")); saveSetting("smtp_user", r.FormValue("smtp_user")); saveSetting("smtp_pass", r.FormValue("smtp_pass")); redirectWithMsg(w, r, "/admin", "Msg_Saved") }
func adminUsersHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { return }; var users []User; db.Preload("Items").Find(&users); data := getPageData(u, "users", r.URL.Query().Get("msg")); data.Users = users; tmpl.Execute(w, data) }
func adminResetPwdHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { return }; db.Model(&User{}).Where("id = ?", r.FormValue("id")).Updates(map[string]interface{}{"password": "123456", "force_reset": true}); redirectWithMsg(w, r, "/admin/users", "Msg_Reset") }
func adminReset2FAHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { return }; db.Model(&User{}).Where("id = ?", r.FormValue("id")).Updates(map[string]interface{}{"chat_id": "", "email": ""}); redirectWithMsg(w, r, "/admin/users", "Msg_2FACleared") }
func adminDelUserHandler(w http.ResponseWriter, r *http.Request, u User) { if u.Role != "admin" { return }; db.Delete(&User{}, r.FormValue("id")); redirectWithMsg(w, r, "/admin/users", "Msg_UserDel") }
func adminSimulateHandler(w http.ResponseWriter, r *http.Request, u User) {
	if u.Role != "admin" { return }; s := getSettingsMap(); sent := false
	alerts := []string{
		fmt.Sprintf("• [%s] <b>example.com</b> (7 %s)", LangMap[u.Language]["Sim_Domain"], LangMap[u.Language]["Sim_DaysLeft"]),
		fmt.Sprintf("• [%s] <b>HK-Node-01</b> (%s)", LangMap[u.Language]["Sim_Server"], LangMap[u.Language]["Sim_Expired"]),
	}
	msgBody := fmt.Sprintf("%s<br><br><i>🛡️ ExpiryGuard System</i>", strings.Join(alerts, "<br>"))
	tgMsg := fmt.Sprintf("⚠️ <b>%s (Simulation)</b><br><br>%s", LangMap[u.Language]["Notify_Title"], msgBody)
	if s["tg_token"] != "" && u.ChatID != "" { sendTg(s["tg_token"], u.ChatID, tgMsg); sent = true }
	if s["smtp_host"] != "" && u.Email != "" { sendEmail(s, u.Email, "[SIMULATION] "+LangMap[u.Language]["Notify_Title"], msgBody); sent = true }
	if sent { redirectWithMsg(w, r, "/admin", "Msg_Sent") } else { redirectWithMsg(w, r, "/admin", "Msg_Fail") }
}
func adminBackupHandler(w http.ResponseWriter, r *http.Request, u User) {
	if u.Role != "admin" { return }
	w.Header().Set("Content-Disposition", "attachment; filename=data.db")
	w.Header().Set("Content-Type", "application/x-sqlite3")
	http.ServeFile(w, r, "data.db")
}

// ----------------- Utils -----------------
func getSettingsMap() map[string]string { var s []Setting; db.Find(&s); m := make(map[string]string); for _, v := range s { m[v.Key] = v.Value }; return m }
func saveSetting(k, v string) { var s Setting; db.Where("key = ?", k).FirstOrCreate(&s, Setting{Key: k}); db.Model(&s).Update("value", v) }
func sendTg(token, chatID, msg string) { url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token); body := fmt.Sprintf(`{"chat_id":"%s","text":"%s","parse_mode":"HTML"}`, chatID, strings.ReplaceAll(msg, "<br>", "\n")); http.Post(url, "application/json", strings.NewReader(body)) }
func sendEmail(s map[string]string, to, sub, body string) { 
	host, port, u, p := s["smtp_host"], s["smtp_port"], s["smtp_user"], s["smtp_pass"]; if host == "" { return }
	auth := smtp.PlainAuth("", u, p, host)
	htmlBody := fmt.Sprintf(`<div style="background-color:#f3f4f6;padding:20px;font-family:sans-serif;"><div style="max-width:600px;margin:0 auto;background:white;padding:30px;border-radius:12px;box-shadow:0 4px 6px rgba(0,0,0,0.05);"><h2 style="color:#1e293b;margin-top:0;">%s</h2><div style="color:#475569;font-size:16px;line-height:1.6;">%s</div></div></div>`, sub, body)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", u, to, sub, htmlBody)
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ServerName: host}; conn, err := tls.Dial("tcp", host+":"+port, tlsConfig); if err != nil { return }; client, _ := smtp.NewClient(conn, host); client.Auth(auth); client.Mail(u); client.Rcpt(to); w, _ := client.Data(); w.Write([]byte(msg)); w.Close(); client.Quit() 
}
func startScheduler() { for range time.NewTicker(1 * time.Minute).C { nowUTC := time.Now().UTC(); if nowUTC.Minute() != 0 { continue }; s := getSettingsMap(); if s["tg_token"] == "" { continue }; var users []User; db.Find(&users); for _, u := range users { loc, err := time.LoadLocation(u.Timezone); if err != nil { loc, _ = time.LoadLocation("Asia/Shanghai") }; localTime := nowUTC.In(loc); if localTime.Hour() == u.NotifyTime { var items []Item; db.Where("user_id = ?", u.ID).Find(&items); var alerts []string; for _, it := range items { t, _ := time.ParseInLocation("2006-01-02", it.Date, loc); days := int(t.Sub(localTime).Hours() / 24); if days == 7 || days == 3 || days == 1 || days == 0 { alerts = append(alerts, fmt.Sprintf("• [%s] <b>%s</b> (%dd)", it.Category, it.Name, days)) } }; if len(alerts) > 0 { msgBody := fmt.Sprintf("%s<br><br><i>🛡️ ExpiryGuard System</i>", strings.Join(alerts, "<br>")); tgMsg := fmt.Sprintf("⚠️ <b>%s</b><br><br>%s", LangMap[u.Language]["Notify_Title"], msgBody); if u.ChatID != "" { sendTg(s["tg_token"], u.ChatID, tgMsg) }; if u.Email != "" { sendEmail(s, u.Email, LangMap[u.Language]["Notify_Title"], msgBody) } } } } } }
