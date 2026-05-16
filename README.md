# Reviz 帳簿

單機 web 版企業/個人記帳，靈感來自 Simpany 雲端帳簿。Go + SQLite，單一 binary，不需資料庫伺服器。

## 功能

- **日記帳**：交易 CRUD，支援收入 / 支出 / 帳戶間轉帳。
- **帳戶總覽**：資產 / 負債分區，自動計算各帳戶餘額與淨值。
- **分類管理**：收入 / 成本 / 費用 / 股東權益 / 其他。
- **專案**：可在交易掛專案，損益表可依專案篩選。
- **損益表**：依年度、月份自動產出（收入 → 成本 → 毛利 → 費用 → 淨利）。
- **儀表板**：四張總覽卡 + 每月收支長條圖 + YTD 淨利折線圖 + 前 5 大費用/收入分類。
- **CSV 匯出 / 匯入**：以名稱對應分類、帳戶、專案。

## 執行

```sh
go build -o reviz-accounting .
./reviz-accounting -create-user alice -create-role owner   # 建第一個帳號
./reviz-accounting                                          # 啟動 web server
```

伺服器預設 `http://localhost:8080`，SQLite 檔在 `data/reviz.db`。

```sh
./reviz-accounting -addr :9000 -db /path/to/mybook.db
```

第一次啟動會自動建表並塞入一組預設分類與三個基本帳戶；可在「設定」頁與「分類 / 帳戶」頁中調整。

## 帳號與角色

三種角色（由高到低權限）：

| 角色 | 可以做的事 |
|---|---|
| `owner` | 所有權限 + 使用者管理 |
| `accountant` | 可記帳：新增/修改交易、帳戶、分類、專案、設定、匯入 CSV |
| `viewer` | 唯讀：看儀表板、日記帳、損益表、帳戶等所有頁面 |

第一個 owner 必須從 CLI 建立：

```sh
./reviz-accounting -create-user alice                          # 預設角色 owner
./reviz-accounting -create-user bob -create-role accountant
```

之後 owner 在 web 上的 `/users` 頁可以增刪、改密碼、改角色、啟用/停用。

Session 用 cookie + DB 存（30 天）。停用使用者會自動把該人的 session 全清掉。改密碼也會踢出其他裝置。

## 編譯

```sh
go build -o reviz-accounting
./reviz-accounting
```

SQLite driver 使用 [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite)（純 Go，無 cgo），可直接交叉編譯。

## 資料模型

| 表 | 用途 |
|---|---|
| `settings` | 公司名稱、會計年度等 key/value |
| `accounts` | 帳戶（資產 / 負債） |
| `categories` | 分類（收入/成本/費用/股東權益/其他） |
| `projects` | 專案 |
| `transactions` | 交易：日期、敘述、分類、金額（正整數 cents）、from_account、to_account、project、備註 |
| `users` | 使用者：username、bcrypt password_hash、role、active、last_login_at |
| `sessions` | Session：id (32-byte base64-url token)、user_id、expires_at、user_agent、ip |

交易方向以 `from_account_id` / `to_account_id` 表達：
- **收入**：只填 `to_account_id`
- **支出**：只填 `from_account_id`
- **轉帳**：兩者皆填

帳戶餘額 = `SUM(amount when to_account=id) - SUM(amount when from_account=id)`。

## CSV 欄位

```
code,date,description,category,amount,from_account,to_account,project,note
```

- `date` 格式 `YYYY-MM-DD`
- `amount` 為正數；方向由 from/to 欄位決定
- 分類/帳戶/專案以**名稱**對應；找不到對應的 account 該筆會被略過

## Docker

純 Go binary，runtime image 用 `distroless/static`，無外部相依：

```sh
docker build -t reviz-accounting .
docker run --rm -p 8080:8080 -v "$PWD/data:/data" reviz-accounting

# 建第一個帳號（互動輸入密碼）
docker run --rm -it -v "$PWD/data:/data" reviz-accounting -create-user alice
```

Container 內預設讀寫 `/data/reviz.db`，所以 `-v <host>:/data` 把資料目錄掛進去。

## Cloud Run 部署

整個 pipeline 在 `cloudbuild.yaml`：build image → push 到 Artifact Registry → 部署到 Cloud Run。SQLite 持久化用 GCS bucket 掛到 `/data`，service 鎖死單一實例（min=max=1）避免 SQLite 多 writer locking 問題。

### 一次性 setup

```sh
PROJECT_ID=<your-project>
REGION=asia-east1
REPO=reviz
BUCKET=$PROJECT_ID-reviz-data

# Artifact Registry repo
gcloud artifacts repositories create $REPO \
  --repository-format=docker --location=$REGION

# GCS bucket for SQLite
gcloud storage buckets create gs://$BUCKET --location=$REGION

# Cloud Build SA 需要的 roles
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')
SA=$PROJECT_NUMBER-compute@developer.gserviceaccount.com
for role in run.admin iam.serviceAccountUser storage.objectAdmin artifactregistry.writer; do
  gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$SA --role=roles/$role
done
```

### 觸發 build

```sh
gcloud builds submit \
  --substitutions=_REGION=$REGION,_REPO=$REPO,_BUCKET=$BUCKET
```

或在 Cloud Build console 設 GitHub trigger 指到這個 repo，每次推 main 自動 deploy。

### 建第一個帳號

Cloud Run 沒有 SSH，建議用 [`gcsfuse`](https://cloud.google.com/storage/docs/gcsfuse-install) 把 bucket 掛到本機後跑 CLI：

```sh
mkdir -p /tmp/reviz-bucket
gcsfuse $BUCKET /tmp/reviz-bucket
./reviz-accounting -create-user alice -db /tmp/reviz-bucket/reviz.db
fusermount -u /tmp/reviz-bucket   # macOS: umount /tmp/reviz-bucket
```

之後到 Cloud Run service URL 用 `alice` 登入即可。owner 可以從 `/users` 頁面繼續加人。

### Cloud Run 限制注意

- **單一實例**：因為 SQLite。如果未來要多實例，需要遷移到 Postgres（Cloud SQL）。
- **GCS volume locking**：Cloud Run 用 gcsfuse 掛 GCS，SQLite 的 advisory file locking 不完全支援。`min=max=1` 是避免 race 的關鍵。
- **冷啟動**：min-instances=1 表示 24/7 都有實例在跑，每月約 ~$10 USD CPU/memory 成本（512Mi / 1 vCPU）。要更省可改成 min=0，但 SQLite + GCS 在 cold-start 時可能有 lock contention。

## 設計筆記

公式邏輯反推自 Simpany 雲端帳簿 v0.4.0 Excel 範本。原檔的損益表/儀表板大量使用 Google Sheets 的 QUERY / ARRAYFORMULA，在 Excel 匯出時會被凍結成快取常數。本實作改寫為 SQL `GROUP BY` 查詢，跑得快也好維護。

## 之後可以加（單人版尚未實作）

- 列印用樣式 (`@media print`)
- 多公司/多帳本
- 多年度結轉
- 圖表更細的下鑽
- 編輯彈窗 / inline 編輯 (HTMX 強化)
