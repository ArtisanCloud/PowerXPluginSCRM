SYNC_LIFECYCLE_SRC := docs/lifecycle
SYNC_LIFECYCLE_DEST := docs/integration/01_plugin_lifecycle

.PHONY: sync-lifecycle-docs
sync-lifecycle-docs:
	@echo "[docs] Syncing $(SYNC_LIFECYCLE_SRC) -> $(SYNC_LIFECYCLE_DEST)"
	@mkdir -p $(SYNC_LIFECYCLE_DEST)
	@rsync -a $(SYNC_LIFECYCLE_SRC)/ $(SYNC_LIFECYCLE_DEST)/
