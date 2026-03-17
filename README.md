# Investment Dashboard

모노레포 구조의 투자 대시보드 웹서비스.

## Quick Start

```bash
# 1. 환경변수 설정
cp server/.env.example server/.env
# .env 파일에서 API 키 설정

# 2. 의존성 설치
make install

# 3. 개발 서버 시작
make dev
```

- Server: http://localhost:8080
- Client: http://localhost:5173

## Tech Stack

- **Frontend**: React 19, TypeScript, TailwindCSS v4, TradingView Lightweight Charts
- **Backend**: Go 1.24, Chi router, gorilla/websocket
- **AI**: Anthropic Claude / OpenAI GPT (configurable)
- **Data**: Yahoo Finance API
