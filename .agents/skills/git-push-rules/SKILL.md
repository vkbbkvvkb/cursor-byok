---
name: git-push-rules
description: 本仓库的 git 推送约束：只推送到用户的 fork（myrepo，https://github.com/vkbbkvvkb/cursor-byok），绝不写入主线 leookun/cursor-byok（origin）。执行 git push 相关操作前先阅读本规则。
---

# Git 推送规则

## 硬性约束

- **只推送到 fork**：remote `myrepo` = `https://github.com/vkbbkvvkb/cursor-byok.git`
- **绝不写入主线**：remote `origin` = `https://github.com/leookun/cursor-byok.git`，`origin` 只允许 fetch/pull，不允许 push。
- 本地账号 `vkbbkvvkb` 对 `leookun/cursor-byok` 无写权限，向 `origin` push 必然返回 403，不要尝试。

## Git 配置

- `branch.main.pushRemote` 已设为 `myrepo`，因此不带 remote 的 `git push` 默认推送到 fork。
- `branch.main.remote`（upstream）保持 `origin`，因此 `git pull` / `git fetch` 仍从主线获取更新。

## 约定

- 需要推送时，显式使用 `git push myrepo <branch>`，或直接 `git push`（默认走 myrepo）。
- 需要同步主线更新时，用 `git fetch origin` + `git merge origin/main`（或 `git pull origin main`），**不要** `git push origin`。
- 如 fork 与本地历史分叉，优先用 merge（保留双方历史）而非 force push；确实需要 force push 时，先确认不会丢失 fork 上的提交并征得用户同意。
- 若用户提出新的推送目标或改动 remote 配置，以最新用户指示为准，并同步更新本规则。