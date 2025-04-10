# playlist-exporter-spotify-to-YTM

A Go-based CLI tool to migrate your Spotify playlists or liked songs to YouTube as private playlists. It supports:

- 🔐 OAuth login for both Spotify and YouTube
- 🎵 Selection of specific playlists or migration of liked songs
- 🧠 Intelligent matching using YouTube Topic uploads
- 🔁 Resume support using `migrated.txt`
- 🛑 Quota control by limiting songs per run

---

## 🔧 Setup

### 1. Clone the repository
```bash
git clone https://github.com/OcheOps/playlist-exporter-spotify-to-YTM.git
cd playlist-exporter-spotify-to-YTM
```

### 2. Create a `.env` file
```env
SPOTIFY_ID=your_spotify_client_id
SPOTIFY_SECRET=your_spotify_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
```

You can get these credentials from:
- [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
- [Google Cloud Console](https://console.cloud.google.com/apis/credentials)

Set the redirect URIs:
- Spotify: `http://localhost:8080/spotify-callback`
- Google: `http://localhost:8090/ytcallback`

### 3. Install Go dependencies
```bash
go mod tidy
```

---

## ▶️ Run the tool
```bash
go run main.go
```

Follow the prompts to:
- Login to Spotify and YouTube
- Choose a playlist or migrate liked songs

---

## 🧠 Features

- **Smart Matching:** Uses `track name + artist + topic` to find official YouTube uploads
- **Resume Support:** Tracks are saved in `migrated.txt` to avoid duplicates
- **Quota Safe:** Limits each run to 100 tracks (customizable in `main.go`)
- **Fallback Logic:** If no `- Topic` upload is found, uses the first search result

---

## 🐞 Known Issues

### Unofficial YouTube Music API Limitations
- Initially, we attempted to use `ytmusicapi`, the unofficial YouTube Music API, for better search accuracy.
- However, recent changes to YouTube Music have made this API unreliable and difficult to authenticate without browser header scraping.

**Solution:**
- We reverted to using the official YouTube Data API with smart query construction (e.g., adding "topic") for better results.


### YouTube Data API Quota Limit
- YouTube has a daily quota of 10,000 units.
- Each `Search.list` costs **100 units**, and each `PlaylistItems.insert` costs **50 units**.
- With large playlists or liked songs (e.g., 5,000+ tracks), the quota will likely be exceeded.

**Solution:**
- The tool limits to 100 songs per run (configurable).
- You can re-run the script daily — already migrated songs are skipped using `migrated.txt`.

## 💡 Tips

- Want to migrate more songs? Just run it again later — it skips already added tracks.
- Adjust `maxToTransfer` if you want to process fewer tracks.

---

## ✅ Todo / Coming Soon

- [ ] Add support for Spotify → YouTube Music 
- [ ] Add support for YouTube Music → Spotify
- [ ] Add support for other platforms (Apple Music, Deezer, etc.)
- [ ] Build a web/desktop UI for non-technical users
- [ ] Add CLI flags for `--resume`, `--reset`, and `--all`
- [ ] Add support for importing/exporting playlists via file (CSV/JSON)
- [ ] Add caching for search results to reduce API usage
- [ ] Add a logging system to track skipped, failed, and successful matches
- [ ] `--playlist` and `--liked` CLI flags
- [ ] `--limit=N` flag to customize batch size
- [ ] Export to JSON or CSV
- [ ] Add support for YouTube Music → Spotify migration
- [ ] Add support for other platforms (e.g., Apple Music, Deezer)
- [ ] Build a web or desktop UI for non-technical users

---

## ☕ Support Me

If you found this useful and want to say thanks:

[Buy Me a Coffee](https://www.buymeacoffee.com/ochecodes)

---

## 📄 License

MIT License. Feel free to fork, extend, and contribute


