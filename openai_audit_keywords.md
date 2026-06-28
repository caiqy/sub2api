# OpenAI 使用政策 — 审计拦截关键词

> 基于 [OpenAI Usage Policies](https://openai.com/policies/usage-policies/) 和 [Terms of Use](https://openai.com/policies/terms-of-use/) 整理。
> 匹配规则：大小写不敏感子串匹配（`strings.Contains(lower(input), lower(kw))`）。
> 关键词格式：JSON string array，直接可替换 `content_moderation_config.blocked_keywords`。

---

## 1. 儿童性虐待材料 (CSAM / Child Sexual Abuse Material)

> Usage Policy: "Prohibition on Child Sexual Abuse Material" — 禁止使用 OpenAI 服务生成涉及未成年人的性内容（图像、文本或任何格式）。

```json
[
  "萝莉控","正太控","幼女","幼童","男童","女童",
  "幼齿","稚嫩","低龄","未成年色情","儿童色情",
  "小学生裸","中学生裸","小小年纪","小妹妹裸",
  "child porn","csam","cp图片","幼交",
  "lolita sex","underage sex","child sex",
  "pedo","pedophile","child abuse material",
  "preteen","underage nude","teen nude",
  "boylove underage","ageplay minor",
  "幼儿园偷拍","小学偷拍","儿童偷拍",
  "秀兰邓波儿裸","儿童sq","幼女视频"
]
```

## 2. 大规模杀伤性武器 (WMD / Weapons of Mass Destruction)

> Usage Policy: "Prohibition on Weapons of Mass Destruction Assistance" — 禁止协助制造生物、化学、核或放射性武器。

```json
[
  "制造炸弹","制作炸弹","炸弹教程","炸弹配方",
  "土制炸弹","管状炸弹","压力锅炸弹","汽车炸弹",
  "生化武器","化学武器","生物武器","放射性武器",
  "脏弹","核弹","原子弹","中子弹","氢弹",
  "毒气","神经毒气","沙林","芥子气","VX毒剂",
  "蓖麻毒素","肉毒杆菌","炭疽","鼠疫武器",
  "核材料","铀浓缩","钚提取","离心机",
  "WMD","CBRN","chemical weapon","biological weapon",
  "nuclear bomb","dirty bomb","ricin",
  "anthrax","sarin","nerve agent","mustard gas",
  "bomb making","how to make bomb","explosive recipe",
  "硝酸铵炸弹","化肥炸弹","TATP","过氧化丙酮",
  "武器级","大规模杀伤","mass casualty"
]
```

## 3. 网络武器与恶意软件 (Cyberweapons & Malware)

> Usage Policy: "Cyberweapons and Malware Prohibition" — 禁止利用 OpenAI 服务创建网络武器、恶意代码或设计用于对计算机系统造成重大损害的工具。

```json
[
  "勒索病毒","勒索软件","文件加密病毒","赎金病毒",
  "木马程序","远控木马","键盘记录","屏幕监控",
  "僵尸网络","肉鸡","DDoS攻击","CC攻击",
  "漏洞利用","0day","零日漏洞","exp",
  "exploit开发","payload生成","shellcode","webshell",
  "远程控制","RAT","后门程序","反弹shell",
  "免杀","免杀制作","免杀教程","过杀软",
  "钓鱼网站","钓鱼页面","仿冒网站","克隆网站",
  "SQL注入","XSS攻击","CSRF攻击","文件包含",
  "提权","横向移动","内网渗透","域渗透",
  "社工库","拖库","撞库","密码爆破",
  "网银木马","支付劫持","剪贴板劫持",
  "恶意APK","短信拦截木马","手机监控",
  "wifi破解","wifi嗅探","中间人攻击",
  "入侵教程","黑客教程","黑阔教程",
  "ransomware","botnet","trojan","keylogger",
  "exploit kit","zero day","metasploit","empire",
  "cobalt strike","mimikatz","bloodhound",
  "phishing kit","credential harvester","cookie stealer",
  "info stealer","infostealer","stealer log",
  "malware builder","crypter","binder","packer",
  "redline stealer","vidar stealer","raccoon stealer"
]
```

## 4. 选举虚假信息 (Election Disinformation)

> Usage Policy: "Election Disinformation Prohibition" — 禁止利用 OpenAI 服务制造旨在破坏选举的内容。

```json
[
  "选举舞弊","投票造假","选票操纵","暗箱操作",
  "选举干预","操控选举","颠覆选举","破坏选举",
  "虚假候选人","候选人黑料","伪造候选人声明",
  "投票站攻击","选票销毁","恐吓选民",
  "冒充选举机构","选举结果否认","不承认选举",
  "民主诈骗","选举政变","颜色革命",
  "vote rigging","election fraud","ballot stuffing",
  "fake candidate","voter suppression","election interference",
  "deepfake politician","synthetic candidate",
  "election disinformation","vote manipulation",
  "伪造选票","虚假选举声明","选举舞弊证据"
]
```

## 5. 非法监控与跟踪 (Unlawful Surveillance)

> Usage Policy: "Unlawful Surveillance and Tracking Prohibition" — 禁止建立侵犯公民权利的监控工具。

```json
[
  "跟踪软件","间谍软件","手机监控","通话录音",
  "定位跟踪","GPS跟踪","实时定位","行踪监控",
  "偷拍设备","针孔摄像头","隐蔽拍摄","偷录",
  "窃听器","窃听设备","电话窃听","房间窃听",
  "人脸识别绕过","身份冒用","冒充他人",
  "开盒","人肉搜索","社工查询","户籍查询",
  "手机定位","查开房记录","查通话记录",
  "查微信聊天","查QQ聊天","查短信记录",
  "stalkerware","spouseware","spyware",
  "location tracking","phone monitoring","keystroke logger",
  "facial recognition bypass","surveillance tool",
  "非法监控","违法监听","侵犯隐私",
  "查身份证信息","查学籍信息","查快递信息"
]
```

## 6. 经济损害与金融欺诈 (Economic Harm / Financial Fraud)

> Usage Policy: 禁止造成经济损害的活动——金融欺诈、市场操纵、欺骗性内容。

```json
[
  "洗钱","跑分","洗黑钱","地下钱庄",
  "信用卡盗刷","CVV","料子","轨道料",
  "银行账户买卖","对公账户","公司账户买卖",
  "刷单","虚假交易","刷信誉","空包",
  "传销","庞氏骗局","资金盘","拆分盘",
  "杀猪盘","电信诈骗","冒充公检法","冒充客服",
  "虚假投资","高回报投资","稳赚不赔","躺着赚钱",
  "非法集资","金融诈骗","贷款诈骗","套路贷",
  "套现","花呗套现","白条套现","信用卡套现",
  "卖微信","卖QQ","卖支付宝","买卖账号",
  "实名认证绕过","人脸认证绕过","跳过KYC",
  "假发票","假合同","假公章","PS印章",
  "money laundering","cvv shop","carding",
  "ponzi scheme","pyramid scheme","pig butchering",
  "romance scam","investment fraud","boiler room",
  "fake identity","synthetic identity","identity theft",
  "financial fraud","wire fraud","bank fraud"
]
```

## 7. 自残与自杀 (Self-Harm / Suicide)

> OpenAI Moderation API: "self-harm" / "self-harm/intent" / "self-harm/instructions"

```json
[
  "自杀","不想活","活不下去","死了算了","生无可恋",
  "自杀方法","怎么自杀","如何自杀","自杀教程",
  "自杀手册","无痛自杀","安乐死教程","安眠药自杀",
  "跳楼","上吊","割腕","割喉","烧炭自杀",
  "卧轨","跳河","投江","跳海","自焚",
  "一了百了","解脱","永远睡着","离开这个世界",
  "自我了断","结束生命","不想存在","消失",
  "遗书","绝笔","最后的话","永别",
  "自残","伤害自己","割自己","烫自己",
  "厌食","催吐","绝食","暴食","吃吐",
  "割肉","划手","伤口感染","失血过多",
  "suicide method","how to kill myself","suicide instructions",
  "self harm","cutting yourself","eating disorder",
  "ana tips","mia tips","pro ana","pro mia",
  "thinspiration","thinspo","meanspo","bonespo",
  "starving myself","purging","binge purge",
  "how to die","best way to die","painless death",
  "kill yourself","kms","end my life"
]
```

## 8. 仇恨言论 (Hate Speech)

> OpenAI Moderation API: "hate" / "hate/threatening"

```json
[
  "种族歧视","种族优越","种族灭绝","劣等民族",
  "法西斯","纳粹","希特勒","万字旗","第三帝国",
  "白人至上","KKK","三K党","种族清洗",
  "支那","东亚病夫","黄祸","chink",
  "黑鬼","nigger","negro","猩猩",
  "台独","藏独","疆独","港独","蒙独",
  "辱华","反华","排华","灭华","亡国",
  "精日","汉奸","卖国贼","走狗",
  "同性恋变态","恐同","反同","烧死同性恋",
  "残疾人该死","残废","畸胎","智障该死",
  "宗教歧视","邪教徒","异教徒该杀","亵渎神明",
  "屠犹","反犹","犹太人控制","犹太阴谋",
  "hate speech","white supremacy","racial superiority",
  "ethnic cleansing","genocide","final solution",
  "islamophobia","antisemitic","xenophobia",
  "transphobia","homophobia","ableism",
  "hate crime","race war","ethnostate"
]
```

## 9. 骚扰与威胁 (Harassment / Threats)

> OpenAI Moderation API: "harassment" / "harassment/threatening"

```json
[
  "短信轰炸","电话轰炸","呼死你","疯狂呼叫",
  "弄死你","杀你全家","灭你全家","砍死你",
  "卸你一条腿","打断你的腿","废了你","搞残你",
  "找你麻烦","上门找你","找到你家","知道你住哪",
  "曝光你","人肉你","让你出名","社死你",
  "网暴","网络暴力","集体网暴","围攻",
  "人身攻击","辱骂","谩骂","侮辱",
  "造谣","抹黑","诽谤","污蔑",
  "跟踪你","尾随","蹲点","堵你",
  "死亡威胁","恐怖威胁","炸弹威胁",
  "doxx","swatting","death threat",
  "bomb threat","mass report","cancel",
  "cyberbullying","online harassment","doxxing"
]
```

## 10. 性内容 (Sexual Content)

> OpenAI Moderation API: "sexual" / "sexual/minors"

```json
[
  "色情小说","黄文","小黄文","成人文学",
  "黄片","AV","成人视频","色情片","三级片",
  "裸聊","聊骚","文爱","语爱","磕炮",
  "约炮","一夜情","炮友","床伴",
  "招嫖","嫖娼","卖淫","外围","楼凤",
  "包养","求包养","找干爹","糖爹","sugar daddy",
  "性服务","上门服务","全套服务","莞式",
  "迷奸","轮奸","强暴","性侵","侵犯",
  "偷拍换衣","偷拍洗澡","偷拍裙底",
  "报复色情","泄露私密照","泄露裸照",
  "deepfake色情","AI换脸色情","合成色情",
  "性交易","肉体交易","皮肉生意",
  "porn","xxx","adult video","sex video",
  "escort","prostitution","sex work",
  "non consensual","revenge porn","leaked nudes",
  "upskirt","voyeur","hidden camera sex",
  "incest","necrophilia","bestiality"
]
```

## 11. 暴力与血腥 (Violence / Graphic Violence)

> OpenAI Moderation API: "violence" / "violence/graphic"

```json
[
  "虐杀","肢解","分尸","斩首","碎尸",
  "强奸杀人","奸杀","先奸后杀","虐尸",
  "活体解剖","活剥","剥皮","抽筋",
  "挖眼","割舌","拔牙","拔指甲",
  "烹尸","煮尸","吃人","食人","人吃人",
  "血祭","献祭","活人祭","祭品",
  "恐怖袭击","独狼袭击","大规模枪击",
  "校园枪击","无差别攻击","随机杀人",
  "酷刑","水刑","电刑","凌迟","炮烙",
  "屠杀","大屠杀","血洗","屠城",
  "凶杀现场","血腥图片","尸检照片",
  "torture","mutilation","dismemberment",
  "beheading","execution","mass shooting",
  "gore","snuff","cartel execution",
  "terror attack","school shooting","active shooter",
  "cannibalism","ritual killing","human sacrifice"
]
```

## 12. 冒充与欺骗 (Impersonation / Deception)

```json
[
  "冒充身份","伪造身份","假扮","假冒",
  "伪造文件","伪造证件","伪造学历","伪造执照",
  "虚假新闻","假新闻","编造新闻","捏造",
  "深度伪造","deepfake","AI生成假视频",
  "冒充名人","冒充官员","冒充执法","冒充律师",
  "伪造签名","仿冒签名","电子签名伪造",
  "虚假认证","蓝V伪造","企业认证伪造",
  "fake news","misinformation","disinformation",
  "impersonation","identity fraud","synthetic media",
  "fabricated evidence","doctored image","manipulated video"
]
```

## 13. 赌博 (Gambling)

> Usage Policy: 禁止推广赌博活动。

```json
[
  "在线赌博","网上赌场","真人视讯","百家乐",
  "老虎机","轮盘","二十一点","德州扑克",
  "六合彩","香港六合彩","澳门赌场","赌球",
  "时时彩","快三","快乐8","福彩3D",
  "体育博彩","赔率","盘口","让球",
  "棋牌游戏","捕鱼游戏","电玩城",
  "赌资","赌注","筹码","上分","下分",
  "刷水","套利","对刷","刷流水",
  "online casino","sports betting","slot machine",
  "gambling site","betting odds","fixed match",
  "casino bonus","free spins","no deposit bonus"
]
```

## 14. 毒品与管制物质 (Drugs / Controlled Substances)

```json
[
  "毒品","贩毒","买毒品","卖毒品","制毒",
  "冰毒","海洛因","可卡因","摇头丸","K粉",
  "大麻","麻古","麻果","神仙水","笑气",
  "吸毒","溜冰","嗑药","飞叶子",
  "吸毒工具","冰壶","溜冰壶","注射器",
  "毒品交易","拿货","出货","散货",
  "罂粟","大麻种植","毒品种植",
  "drug dealer","buy drugs","methamphetamine",
  "cocaine","heroin","weed delivery",
  "darknet market","silk road","drug marketplace"
]
```

## 15. 合规与安全绕过 (Policy Evasion / Jailbreak)

```json
[
  "越狱提示词","jailbreak prompt","绕过安全",
  "绕过审核","绕过限制","解除限制",
  "角色扮演越狱","DAN prompt","开发者模式",
  "忽略指令","忽略规则","忽略之前",
  "假装你是","你现在是","扮演",
  "system prompt泄露","prompt injection",
  "ignore all instructions","ignore previous",
  "disregard guidelines","bypass filter",
  "pretend you are","you are now","act as",
  "override safety","disable content filter",
  "token smuggling","prompt leaking","prompt extraction"
]
```

---

## 完整合并 JSON

以下为全部关键词合并后的 JSON 数组，可直接用于替换 `content_moderation_config.blocked_keywords`。

> ⚠️ 注意：OpenAI 本身使用深度学习模型（语义级别）而非简单关键词匹配来检测违规内容。本关键词列表用于 sub2api 的**第一层快速拦截**，作为 OpenAI 审核 API 的补充，不能替代 OpenAI 审核 API。

```json
[
  "儿童色情","未成年色情","幼女","幼童","幼交","child porn","csam","pedo",
  "制造炸弹","制作炸弹","生化武器","化学武器","核弹","WMD","bomb making",
  "勒索病毒","勒索软件","木马程序","远控","漏洞利用","0day","exploit",
  "webshell","免杀","钓鱼网站","SQL注入","拖库","ransomware","botnet",
  "选举舞弊","投票造假","选举干预","election fraud","voter suppression",
  "跟踪软件","间谍软件","手机监控","开盒","人肉搜索","stalkerware","spyware",
  "洗钱","信用卡盗刷","CVV","刷单","杀猪盘","电信诈骗","money laundering",
  "自杀","不想活","自杀方法","自残","suicide method","self harm",
  "厌食","催吐","eating disorder","pro ana","怎么自杀",
  "种族歧视","法西斯","纳粹","支那","台独","藏独","疆独","港独",
  "nigger","chink","white supremacy","hate speech",
  "短信轰炸","呼死你","杀你全家","弄死你","网暴","doxx","death threat",
  "黄文","黄片","约炮","招嫖","卖淫","包养","porn","escort","prostitution",
  "迷奸","轮奸","强暴","偷拍裙底","revenge porn","non consensual",
  "虐杀","肢解","分尸","斩首","吃人","酷刑","torture","mutilation","gore",
  "deepfake","伪造身份","虚假新闻","fake news","impersonation",
  "在线赌博","网上赌场","赌球","gambling","online casino",
  "毒品","贩毒","冰毒","海洛因","吸毒","drug","cocaine","heroin",
  "jailbreak","越狱提示词","绕过审核","忽略指令","DAN prompt",
  "抓包","逆向","接码平台","人脸绕过","实名绕过","脱壳",
  "破解教程","破解补丁","注册机","激活码生成","序列号生成",
  "去除验证","绕过授权","frida","ida pro","ollydbg","x64dbg",
  "去广告","内购破解","vip破解","会员破解","引流破解",
  "so注入","dll注入","内存修改","ce修改器","变速齿轮","gg修改器",
  "加密狗模拟","加密锁模拟","加密锁破解",
  "割腕","烧炭自杀","吃安眠药自杀","跳楼","上吊","遗书",
  "不想活","烧炭","一了百了",
  "萝莉","正太","幼齿","学生妹",
  "买枪","制毒","贩毒","买毒品","洗钱","卖银行卡","卖微信号",
  "网赌","赌博平台","支那","东亚病夫","亡国奴","汉奸","卖国贼",
  "台独","藏独","疆独","港独","法西斯","纳粹","种族灭绝",
  "杀你全家","弄死你","人肉","开盒",
  "短信轰炸","电话轰炸","fake identity","synthetic identity"
]
```

---

## 使用方式

1. 复制上方 "完整合并 JSON" 数组
2. 在 sub2api 后台 → 风控管理 → 内容审核配置 → `blocked_keywords` 粘贴
3. 测试关键词是否命中预期输入（大小写不敏感子串匹配）
