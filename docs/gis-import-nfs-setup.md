# GIS shapefile import — shared NFS (multi-server Coolify)

When **idx-api-web** runs on **re-db** and **re-node-02** with two domains, Coolify routes:

| Hostname | Coolify instance | Host |
|----------|------------------|------|
| `idx.quantyralabs.cc` | https-0 | **re-db** |
| `upload.idx.quantyralabs.cc` | https-1 | **re-node-02** |

Uploads land on re-node-02 disk; **idx-api-worker 1** on re-db often consumes `default` queue jobs and cannot see files on re-node-02's local bind mount.

**Fix:** export `/data/coolify/gis-imports` from one host over **Tailscale** and mount it on the other at the **same path**, then keep Coolify volume mounts unchanged.

Container user: **`nobody`** (uid **65534**, gid **65534**) — see `Dockerfile` `api` / `worker` targets.

---

## Topology (Quantyra production)

| Role | Server | Tailscale IP | Public IP |
|------|--------|--------------|-----------|
| NFS server (export) | re-db (Coolify host) | *(run `tailscale ip -4` on re-db)* | `208.87.128.115` |
| NFS client (mount) | re-node-02 | `100.89.130.19` | — |

Use Tailscale IPs in `/etc/exports` and `/etc/fstab` — **never** expose NFS on the public internet.

Coolify apps that need the mount:

| App | UUID |
|-----|------|
| idx-api-web | `bty6eqpssq65nhsywj1xbfvf` |
| idx-api-worker 1 | `odswimcjslsv86tq2z1mpyoa` |

On **each server** where those apps run, Coolify storage:

- **Source (host):** `/data/coolify/gis-imports`
- **Destination (container):** `/var/cache/geoidx/gis-imports`

---

## 1. NFS server — re-db

SSH to **re-db** as root.

```bash
# Install server
apt-get update && apt-get install -y nfs-kernel-server

# Export directory (keep path identical on both hosts)
mkdir -p /data/coolify/gis-imports
chown 65534:65534 /data/coolify/gis-imports
chmod 775 /data/coolify/gis-imports

# Tailscale IPs — adjust if your mesh differs
RE_NODE_02_TS=100.89.130.19
RE_DB_TS=$(tailscale ip -4)

# Export to re-node-02 only (sync + no_root_squash avoids nobody→nobody mapping issues)
cat >> /etc/exports <<EOF
/data/coolify/gis-imports ${RE_NODE_02_TS}(rw,sync,no_subtree_check,no_root_squash)
EOF

exportfs -ra
systemctl enable --now nfs-server
exportfs -v
```

Optional: if uploads already exist on re-node-02 local disk, copy them to re-db **before** switching re-node-02 to NFS client:

```bash
# From re-node-02, once re-db export is up:
# rsync -av /data/coolify/gis-imports/ root@${RE_DB_TS}:/data/coolify/gis-imports/
```

---

## 2. NFS client — re-node-02

SSH to **re-node-02** as root.

```bash
apt-get update && apt-get install -y nfs-common

# Get re-db Tailscale IP (run on re-db: tailscale ip -4)
RE_DB_TS=<re-db-tailscale-ip>

# If local data exists, move aside once (after rsync to server if needed)
mv /data/coolify/gis-imports /data/coolify/gis-imports.local.bak 2>/dev/null || true
mkdir -p /data/coolify/gis-imports

# Persistent mount (Tailscale only)
grep -q gis-imports /etc/fstab || echo "${RE_DB_TS}:/data/coolify/gis-imports /data/coolify/gis-imports nfs4 rw,hard,timeo=600,retrans=2,_netdev,nofail 0 0" >> /etc/fstab

mount -a
mount | grep gis-imports
touch /data/coolify/gis-imports/.nfs-test && rm /data/coolify/gis-imports/.nfs-test
chown 65534:65534 /data/coolify/gis-imports
```

Verify from re-node-02:

```bash
RE_DB_TS=<re-db-tailscale-ip>
echo nfs-client-test | tee /data/coolify/gis-imports/.nfs-client-test
ssh root@${RE_DB_TS} 'cat /data/coolify/gis-imports/.nfs-client-test && rm /data/coolify/gis-imports/.nfs-client-test'
```

---

## 3. Coolify — confirm volumes (both hosts)

For **idx-api-web** and **idx-api-worker 1**, on **re-db** and **re-node-02** instances:

1. **Storages** → add or verify bind mount:
   - Host: `/data/coolify/gis-imports`
   - Container: `/var/cache/geoidx/gis-imports`
2. **Redeploy** web and worker 1 on **both** servers after NFS is live.

Env (already set):

```env
GIS_IMPORT_PATH=/var/cache/geoidx/gis-imports
GIS_IMPORT_MAX_BYTES=536870912
```

---

## 4. Verify end-to-end

From repo on any machine with DB access:

```bash
./scripts/verify-gis-import-nfs.sh
```

Or manually:

1. Upload `Parcels.zip` on dashboard → pinellas source.
2. On **re-db** and **re-node-02**:

   ```bash
   ls -lh /data/coolify/gis-imports/pinellas/Parcels.zip
   ```

   Both must show the same size (~118 MB) within seconds.

3. Worker 1 logs — no `upload file not found`; expect ogr2ogr / import progress.

4. DB:

   ```sql
   SELECT id, status, LEFT(error, 80) FROM gis_import_uploads ORDER BY id DESC LIMIT 3;
   ```

   Latest row should reach `done` (large imports may take several minutes).

---

## 5. Failure modes

| Symptom | Likely cause |
|---------|----------------|
| `upload file not found` after NFS | Mount missing on one host; redeploy without volume; stale local dir instead of NFS on re-node-02 |
| `Permission denied` on write | Export dir not `65534:65534` or `root_squash` without `no_root_squash` |
| Mount hangs | Tailscale down; wrong server IP in fstab; firewall blocking NFS (2049) on Tailscale interface |
| File on re-node-02 only | re-node-02 still using local disk — check `mount \| grep gis-imports` shows `nfs4` |
| Import slow | Normal for ~118 MB zip + ogr2ogr; watch worker logs |

---

## Related

- [coolify-env-by-app.md](coolify-env-by-app.md) — `GIS_IMPORT_*` and worker 1 queues
- [gis-sources.md](gis-sources.md) — shapefile upload pipeline
- [coolify-deployment.md](coolify-deployment.md) §8 — multi-DC topology
