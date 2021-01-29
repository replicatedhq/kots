package snapshot

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
)

func Test_parseLogs(t *testing.T) {
	tests := []struct {
		name         string
		logs         string
		wantErrors   []types.SnapshotError
		wantWarnings []types.SnapshotError
		wantHooks    []*types.SnapshotHook
		wantErr      bool
	}{
		{
			name: "basic",
			logs: `time="2020-08-24T15:41:00Z" level=info msg="Setting up backup temp file" backup=velero/qakots-ns24s logSource="pkg/controller/backup_controller.go:494"
time="2020-08-24T15:41:00Z" level=info msg="Setting up plugin manager" backup=velero/qakots-ns24s logSource="pkg/controller/backup_controller.go:501"
time="2020-08-24T15:41:00Z" level=info msg="Getting backup item actions" backup=velero/qakots-ns24s logSource="pkg/controller/backup_controller.go:505"
time="2020-08-24T15:41:00Z" level=info msg="Setting up backup store" backup=velero/qakots-ns24s logSource="pkg/controller/backup_controller.go:511"
time="2020-08-24T15:41:00Z" level=info msg="Writing backup version file" backup=velero/qakots-ns24s logSource="pkg/backup/backup.go:213"
time="2020-08-24T15:41:00Z" level=info msg="Including namespaces: lonesomepod, test" backup=velero/qakots-ns24s logSource="pkg/backup/backup.go:219"
time="2020-08-24T15:41:00Z" level=info msg="Excluding namespaces: <none>" backup=velero/qakots-ns24s logSource="pkg/backup/backup.go:220"
time="2020-08-24T15:41:00Z" level=info msg="Including resources: *" backup=velero/qakots-ns24s logSource="pkg/backup/backup.go:223"
time="2020-08-24T15:41:00Z" level=info msg="Excluding resources: <none>" backup=velero/qakots-ns24s logSource="pkg/backup/backup.go:224"
time="2020-08-24T15:41:15Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:15Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=pods
time="2020-08-24T15:41:15Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=pods
time="2020-08-24T15:41:15Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=pods
time="2020-08-24T15:41:15Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=replicated-deployment-redis-ha-service-test namespace=lonesomepod resource=pods
time="2020-08-24T15:41:15Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=replicated-deployment-redis-ha-service-test namespace=lonesomepod resource=pods
time="2020-08-24T15:41:15Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:15Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Retrieved 7 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql-0 namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=example-mysql-0 namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Adding pvc datadir-example-mysql-0 to additionalItems" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:67" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=datadir-example-mysql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:17Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=datadir-example-mysql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:17Z" level=info msg="Executing PVCAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/backup_pv_action.go:49" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=pvc-dc0e3ccd-8db9-4f1a-a8fa-144e57b5a5ca namespace= resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=info msg="Executing takePVSnapshot" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:400" name=pvc-dc0e3ccd-8db9-4f1a-a8fa-144e57b5a5ca namespace= resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=info msg="Skipping snapshot of persistent volume because volume is being backed up with restic." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:418" name=pvc-dc0e3ccd-8db9-4f1a-a8fa-144e57b5a5ca namespace= persistentVolume=pvc-dc0e3ccd-8db9-4f1a-a8fa-144e57b5a5ca resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=warning msg="Volume datadir in pod test/example-mysql-0 is a hostPath volume which is not supported for restic backup, skipping" backup=velero/qakots-ns24s group=v1 logSource="pkg/restic/backupper.go:156" name=example-mysql-0 namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-nginx-5758b958bf-fhsgs namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=example-nginx-5758b958bf-fhsgs namespace=test resource=pods
time="2020-08-24T15:41:17Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Adding pvc dummydata to additionalItems" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:67" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=dummydata namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:17Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=dummydata namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:17Z" level=info msg="Executing PVCAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/backup_pv_action.go:49" pluginName=velero
time="2020-08-24T15:41:17Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=pvc-a69b7585-8615-47b1-b592-07e44aeb18b8 namespace= resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=info msg="Executing takePVSnapshot" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:400" name=pvc-a69b7585-8615-47b1-b592-07e44aeb18b8 namespace= resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=info msg="Skipping snapshot of persistent volume because volume is being backed up with restic." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:418" name=pvc-a69b7585-8615-47b1-b592-07e44aeb18b8 namespace= persistentVolume=pvc-a69b7585-8615-47b1-b592-07e44aeb18b8 resource=persistentvolumes
time="2020-08-24T15:41:17Z" level=warning msg="Volume dummydata in pod test/example-nginx-5758b958bf-fhsgs is a hostPath volume which is not supported for restic backup, skipping" backup=velero/qakots-ns24s group=v1 logSource="pkg/restic/backupper.go:156" name=example-nginx-5758b958bf-fhsgs namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598111940-wkp4k namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=hello-1598111940-wkp4k namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598112000-tzfnb namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=hello-1598112000-tzfnb namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598112060-5qdrb namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=hello-1598112060-5qdrb namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=job-2-oi2n6q-txxlg namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:18Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql-postgresql-0 namespace=test resource=pods
time="2020-08-24T15:41:18Z" level=info msg="running exec hook" backup=velero/qakots-ns24s group=v1 hookCommand="[/bin/bash -c PGPASSWORD=$POSTGRES_PASSWORD pg_dump -U username -d dbname -h 127.0.0.1 > /scratch/backup.sql]" hookContainer=postgresql hookName="<from-annotation>" hookOnError=Fail hookPhase=pre hookSource=annotation hookTimeout="{3m0s}" hookType=exec logSource="pkg/podexec/pod_command_executor.go:124" name=postgresql-postgresql-0 namespace=test resource=pods
time="2020-08-24T15:41:19Z" level=info msg="stdout: " backup=velero/qakots-ns24s group=v1 hookCommand="[/bin/bash -c PGPASSWORD=$POSTGRES_PASSWORD pg_dump -U username -d dbname -h 127.0.0.1 > /scratch/backup.sql]" hookContainer=postgresql hookName="<from-annotation>" hookOnError=Fail hookPhase=pre hookSource=annotation hookTimeout="{3m0s}" hookType=exec logSource="pkg/podexec/pod_command_executor.go:171" name=postgresql-postgresql-0 namespace=test resource=pods
time="2020-08-24T15:41:19Z" level=info msg="stderr: " backup=velero/qakots-ns24s group=v1 hookCommand="[/bin/bash -c PGPASSWORD=$POSTGRES_PASSWORD pg_dump -U username -d dbname -h 127.0.0.1 > /scratch/backup.sql]" hookContainer=postgresql hookName="<from-annotation>" hookOnError=Fail hookPhase=pre hookSource=annotation hookTimeout="{3m0s}" hookType=exec logSource="pkg/podexec/pod_command_executor.go:172" name=postgresql-postgresql-0 namespace=test resource=pods
time="2020-08-24T15:41:19Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=postgresql-postgresql-0 namespace=test resource=pods
time="2020-08-24T15:41:19Z" level=info msg="Executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
time="2020-08-24T15:41:19Z" level=info msg="Adding pvc data-postgresql-postgresql-0 to additionalItems" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:67" pluginName=velero
time="2020-08-24T15:41:19Z" level=info msg="Done executing podAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
time="2020-08-24T15:41:19Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=data-postgresql-postgresql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:19Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=data-postgresql-postgresql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:19Z" level=info msg="Executing PVCAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/backup_pv_action.go:49" pluginName=velero
time="2020-08-24T15:41:19Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= resource=persistentvolumes
time="2020-08-24T15:41:19Z" level=info msg="Executing takePVSnapshot" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:400" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= resource=persistentvolumes
time="2020-08-24T15:41:19Z" level=info msg="label \"topology.kubernetes.io/zone\" is not present on PersistentVolume, checking deprecated label..." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:427" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= persistentVolume=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d resource=persistentvolumes
time="2020-08-24T15:41:19Z" level=info msg="label \"failure-domain.beta.kubernetes.io/zone\" is not present on PersistentVolume" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:430" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= persistentVolume=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d resource=persistentvolumes
time="2020-08-24T15:41:19Z" level=info msg="No volume ID returned by volume snapshotter for persistent volume" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:453" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= persistentVolume=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d resource=persistentvolumes volumeSnapshotLocation=default
time="2020-08-24T15:41:19Z" level=info msg="Persistent volume is not a supported volume type for snapshots, skipping." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:464" name=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d namespace= persistentVolume=pvc-7d20c877-fe86-4457-a743-b1c5b36a1e0d resource=persistentvolumes
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 3 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Skipping item because it's already been backed up." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:163" name=data-postgresql-postgresql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Skipping item because it's already been backed up." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:163" name=datadir-example-mysql-0 namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Skipping item because it's already been backed up." backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:163" name=dummydata namespace=test resource=persistentvolumeclaims
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=persistentvolumes
time="2020-08-24T15:41:22Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:127" resource=persistentvolumes
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=replicationcontrollers
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=replicationcontrollers
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=replicationcontrollers
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=replicationcontrollers
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=replicationcontrollers
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=namespaces
time="2020-08-24T15:41:22Z" level=info msg="Getting namespace" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:184" namespace=lonesomepod resource=namespaces
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=lonesomepod namespace= resource=namespaces
time="2020-08-24T15:41:22Z" level=info msg="Getting namespace" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:184" namespace=test resource=namespaces
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=test namespace= resource=namespaces
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 2 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=default-token-vjbnz namespace=lonesomepod resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=kotsadm-replicated-registry namespace=lonesomepod resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 3 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=default-token-25q5s namespace=test resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=kotsadm-replicated-registry namespace=test resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql namespace=test resource=secrets
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=resourcequotas
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=resourcequotas
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=resourcequotas
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=resourcequotas
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=resourcequotas
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 4 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-nginx namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql-headless namespace=test resource=endpoints
time="2020-08-24T15:41:22Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=events
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Retrieved 4 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=replicated-deployment-redis-ha-service-test.162e3e82f1524314 namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=replicated-deployment-redis-ha-service-test.162e3e83170db314 namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=replicated-deployment-redis-ha-service-test.162e3e8328ca6fc6 namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=replicated-deployment-redis-ha-service-test.162e3e833d1633b3 namespace=lonesomepod resource=events
time="2020-08-24T15:41:22Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 7 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=hello.162e37c6cdb37049 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg.162e3e73f949f947 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg.162e3e74207dfdf7 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg.162e3e74538175e9 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg.162e3e745cf0acc0 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q-txxlg.162e3e7469cde599 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q.162e3e73f8f6de94 namespace=test resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=services
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=services
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=services
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 4 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-nginx namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql-headless namespace=test resource=services
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=example-config namespace=test resource=configmaps
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=default namespace=lonesomepod resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=default namespace=lonesomepod resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Running ServiceAccountAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/service_account_action.go:77" pluginName=velero
time="2020-08-24T15:41:23Z" level=info msg="Done running ServiceAccountAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/service_account_action.go:120" pluginName=velero
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:169" name=default namespace=test resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Executing custom action" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/item_backupper.go:330" name=default namespace=test resource=serviceaccounts
time="2020-08-24T15:41:23Z" level=info msg="Running ServiceAccountAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/service_account_action.go:77" pluginName=velero
time="2020-08-24T15:41:23Z" level=info msg="Done running ServiceAccountAction" backup=velero/qakots-ns24s cmd=/velero logSource="pkg/backup/service_account_action.go:120" pluginName=velero
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=nodes
time="2020-08-24T15:41:23Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:127" resource=nodes
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=limitranges
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=limitranges
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=limitranges
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=limitranges
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=limitranges
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=podtemplates
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=podtemplates
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=podtemplates
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=podtemplates
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=podtemplates
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=apiregistration.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apiregistration.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=apiservices
time="2020-08-24T15:41:23Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=apiregistration.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=apiservices
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:105" resource=deployments
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=deployments
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=deployments
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=deployments
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=deployments
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-nginx namespace=test resource=deployments.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:105" resource=daemonsets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=daemonsets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=daemonsets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=daemonsets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=daemonsets
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:105" resource=controllerrevisions
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=controllerrevisions
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=controllerrevisions
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=controllerrevisions
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 4 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=controllerrevisions
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql-5db69577d5 namespace=test resource=controllerrevisions.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql-856d648bb7 namespace=test resource=controllerrevisions.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql-8f7cc7bb9 namespace=test resource=controllerrevisions.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql-postgresql-6f9f647b5c namespace=test resource=controllerrevisions.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:105" resource=statefulsets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=statefulsets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=statefulsets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=statefulsets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 2 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=statefulsets
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-mysql namespace=test resource=statefulsets.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=postgresql-postgresql namespace=test resource=statefulsets.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:105" resource=replicasets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=replicasets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=replicasets
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=replicasets
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=replicasets
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=apps/v1 logSource="pkg/backup/item_backupper.go:169" name=example-nginx-5758b958bf namespace=test resource=replicasets.apps
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=events.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=events.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=events
time="2020-08-24T15:41:23Z" level=info msg="Skipping resource because it cohabitates and we've already processed it" backup=velero/qakots-ns24s cohabitatingResource1=events cohabitatingResource2=events.events.k8s.io group=events.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:148" resource=events
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/resource_backupper.go:105" resource=horizontalpodautoscalers
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=horizontalpodautoscalers
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=horizontalpodautoscalers
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=horizontalpodautoscalers
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=autoscaling/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=horizontalpodautoscalers
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/resource_backupper.go:105" resource=jobs
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=jobs
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=jobs
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=jobs
time="2020-08-24T15:41:23Z" level=info msg="Retrieved 4 items" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=jobs
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598111940 namespace=test resource=jobs.batch
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598112000 namespace=test resource=jobs.batch
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/item_backupper.go:169" name=hello-1598112060 namespace=test resource=jobs.batch
time="2020-08-24T15:41:23Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=batch/v1 logSource="pkg/backup/item_backupper.go:169" name=job-2-oi2n6q namespace=test resource=jobs.batch
time="2020-08-24T15:41:23Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:23Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=cronjobs
time="2020-08-24T15:41:23Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=cronjobs
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=cronjobs
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=cronjobs
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 1 items" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=cronjobs
time="2020-08-24T15:41:24Z" level=info msg="Backing up item" backup=velero/qakots-ns24s group=batch/v1beta1 logSource="pkg/backup/item_backupper.go:169" name=hello namespace=test resource=cronjobs.batch
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=certificates.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=certificates.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=certificatesigningrequests
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=certificates.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=certificatesigningrequests
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=networkpolicies
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=networkpolicies
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=networkpolicies
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=networkpolicies
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=networking.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=networkpolicies
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=ingresses
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=ingresses
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=ingresses
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=ingresses
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=networking.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=ingresses
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=poddisruptionbudgets
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=poddisruptionbudgets
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=poddisruptionbudgets
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=poddisruptionbudgets
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=poddisruptionbudgets
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=podsecuritypolicies
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=policy/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=podsecuritypolicies
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=clusterrolebindings
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=clusterrolebindings
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=clusterroles
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=clusterroles
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=rolebindings
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=rolebindings
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=rolebindings
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=rolebindings
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=rolebindings
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=roles
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=roles
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=roles
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=roles
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=rbac.authorization.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=roles
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=storage.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=storage.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=volumeattachments
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=storage.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=volumeattachments
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=storage.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=storageclasses
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=storage.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=storageclasses
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=storage.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=storage.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=csinodes
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=storage.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=csinodes
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=storage.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=csidrivers
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=storage.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=csidrivers
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=admissionregistration.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=admissionregistration.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=mutatingwebhookconfigurations
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=admissionregistration.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=mutatingwebhookconfigurations
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=admissionregistration.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=validatingwebhookconfigurations
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=admissionregistration.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=validatingwebhookconfigurations
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=apiextensions.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=apiextensions.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=customresourcedefinitions
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=apiextensions.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=customresourcedefinitions
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=scheduling.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=scheduling.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=priorityclasses
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=scheduling.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:127" resource=priorityclasses
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=leases
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=leases
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=leases
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=leases
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=coordination.k8s.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=leases
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=node.k8s.io/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=node.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=runtimeclasses
time="2020-08-24T15:41:24Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=node.k8s.io/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=runtimeclasses
time="2020-08-24T15:41:24Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=mongodbusers
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=mongodbusers
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=mongodbusers
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=mongodbusers
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=mongodbusers
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=mongodb
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=mongodb
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=mongodb
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=mongodb
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=mongodb
time="2020-08-24T15:41:24Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=opsmanagers
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=opsmanagers
time="2020-08-24T15:41:24Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=opsmanagers
time="2020-08-24T15:41:24Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=opsmanagers
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=mongodb.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=opsmanagers
time="2020-08-24T15:41:25Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=alertmanagers
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=alertmanagers
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=alertmanagers
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=alertmanagers
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=alertmanagers
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=prometheusrules
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=prometheusrules
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=prometheusrules
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=prometheusrules
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=prometheusrules
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=prometheuses
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=prometheuses
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=prometheuses
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=prometheuses
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=prometheuses
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=thanosrulers
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=thanosrulers
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=thanosrulers
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=thanosrulers
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=thanosrulers
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=servicemonitors
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=servicemonitors
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=servicemonitors
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=servicemonitors
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=servicemonitors
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:105" resource=podmonitors
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=podmonitors
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=podmonitors
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=podmonitors
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=monitoring.coreos.com/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=podmonitors
time="2020-08-24T15:41:25Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=httpproxies
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=httpproxies
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=httpproxies
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=httpproxies
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=httpproxies
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=tlscertificatedelegations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=tlscertificatedelegations
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=tlscertificatedelegations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=tlscertificatedelegations
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=projectcontour.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=tlscertificatedelegations
time="2020-08-24T15:41:25Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=volumesnapshotlocations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=volumesnapshotlocations
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=volumesnapshotlocations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=volumesnapshotlocations
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=volumesnapshotlocations
time="2020-08-24T15:41:25Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=backupstoragelocations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=backupstoragelocations
time="2020-08-24T15:41:25Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=backupstoragelocations
time="2020-08-24T15:41:25Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=backupstoragelocations
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=backupstoragelocations
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=podvolumerestores
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=podvolumerestores
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=podvolumerestores
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=podvolumerestores
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=podvolumerestores
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=backups
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=backups
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=backups
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=backups
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=backups
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=restores
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=restores
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=restores
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=restores
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=restores
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=serverstatusrequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=serverstatusrequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=serverstatusrequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=serverstatusrequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=serverstatusrequests
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=schedules
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=schedules
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=schedules
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=schedules
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=schedules
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=deletebackuprequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=deletebackuprequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=deletebackuprequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=deletebackuprequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=deletebackuprequests
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=resticrepositories
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=resticrepositories
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=resticrepositories
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=resticrepositories
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=resticrepositories
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=downloadrequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=downloadrequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=downloadrequests
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=downloadrequests
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=downloadrequests
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:105" resource=podvolumebackups
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=podvolumebackups
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=podvolumebackups
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=podvolumebackups
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=velero.io/v1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=podvolumebackups
time="2020-08-24T15:41:26Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:26Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:105" resource=databases
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=databases
time="2020-08-24T15:41:26Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=databases
time="2020-08-24T15:41:26Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=databases
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=databases.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=databases
time="2020-08-24T15:41:27Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:105" resource=tables
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=tables
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=tables
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=tables
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=tables
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:105" resource=migrations
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=migrations
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=migrations
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=migrations
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=schemas.schemahero.io/v1alpha4 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=migrations
time="2020-08-24T15:41:27Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=tlscertificatedelegations
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=tlscertificatedelegations
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=tlscertificatedelegations
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=tlscertificatedelegations
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=tlscertificatedelegations
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=ingressroutes
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=ingressroutes
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=ingressroutes
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=ingressroutes
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=contour.heptio.com/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=ingressroutes
time="2020-08-24T15:41:27Z" level=info msg="Backing up group" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/group_backupper.go:101"
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=networkpolicies
time="2020-08-24T15:41:27Z" level=info msg="Skipping resource because it cohabitates and we've already processed it" backup=velero/qakots-ns24s cohabitatingResource1=networkpolicies.extensions cohabitatingResource2=networkpolicies.networking.k8s.io group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:148" resource=networkpolicies
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=replicasets
time="2020-08-24T15:41:27Z" level=info msg="Skipping resource because it cohabitates and we've already processed it" backup=velero/qakots-ns24s cohabitatingResource1=replicasets.extensions cohabitatingResource2=replicasets.apps group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:148" resource=replicasets
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=deployments
time="2020-08-24T15:41:27Z" level=info msg="Skipping resource because it cohabitates and we've already processed it" backup=velero/qakots-ns24s cohabitatingResource1=deployments.extensions cohabitatingResource2=deployments.apps group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:148" resource=deployments
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=daemonsets
time="2020-08-24T15:41:27Z" level=info msg="Skipping resource because it cohabitates and we've already processed it" backup=velero/qakots-ns24s cohabitatingResource1=daemonsets.extensions cohabitatingResource2=daemonsets.apps group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:148" resource=daemonsets
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=ingresses
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=lonesomepod resource=ingresses
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=lonesomepod resource=ingresses
time="2020-08-24T15:41:27Z" level=info msg="Listing items" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:226" namespace=test resource=ingresses
time="2020-08-24T15:41:27Z" level=info msg="Retrieved 0 items" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:240" namespace=test resource=ingresses
time="2020-08-24T15:41:27Z" level=info msg="Backing up resource" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:105" resource=podsecuritypolicies
time="2020-08-24T15:41:27Z" level=info msg="Skipping resource because it's cluster-scoped and only specific namespaces are included in the backup" backup=velero/qakots-ns24s group=extensions/v1beta1 logSource="pkg/backup/resource_backupper.go:127" resource=podsecuritypolicies`,
			wantErrors: []types.SnapshotError{},
			wantWarnings: []types.SnapshotError{
				{
					Title: "Volume datadir in pod test/example-mysql-0 is a hostPath volume which is not supported for restic backup, skipping",
				},
				{
					Title: "Volume dummydata in pod test/example-nginx-5758b958bf-fhsgs is a hostPath volume which is not supported for restic backup, skipping",
				},
			},
			wantHooks: []*types.SnapshotHook{
				{
					Name:          "<from-annotation>",
					Namespace:     "test",
					Phase:         "pre",
					PodName:       "postgresql-postgresql-0",
					ContainerName: "postgresql",
					Command:       "[/bin/bash -c PGPASSWORD=$POSTGRES_PASSWORD pg_dump -U username -d dbname -h 127.0.0.1 > /scratch/backup.sql]",
					Stdout:        "",
					Stderr:        "",
					StartedAt:     mustParseTime(t, time.RFC3339, "2020-08-24T15:41:18Z"),
					FinishedAt:    mustParseTime(t, time.RFC3339, "2020-08-24T15:41:19Z"),
					Errors:        nil,
					Warnings:      nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := parseLogs(strings.NewReader(tt.logs))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantErrors) {
				bGot, _ := json.MarshalIndent(got, "", "  ")
				bWant, _ := json.MarshalIndent(tt.wantErrors, "", "  ")
				t.Errorf("parseLogs() (errors) got = %v, want %v", string(bGot), string(bWant))
			}
			if !reflect.DeepEqual(got1, tt.wantWarnings) {
				bGot, _ := json.MarshalIndent(got1, "", "  ")
				bWant, _ := json.MarshalIndent(tt.wantWarnings, "", "  ")
				t.Errorf("parseLogs() (warnings) got = %v, want %v", string(bGot), string(bWant))
			}
			if !reflect.DeepEqual(got2, tt.wantHooks) {
				bGot, _ := json.MarshalIndent(got2, "", "  ")
				bWant, _ := json.MarshalIndent(tt.wantHooks, "", "  ")
				t.Errorf("parseLogs() (hooks) got = %v, want %v", string(bGot), string(bWant))
			}
		})
	}
}

func mustParseTime(t *testing.T, layout, value string) *time.Time {
	time, err := time.Parse(layout, value)
	if err != nil {
		t.Fatalf("Failed to parse time %s: %v", value, err)
	}
	return &time
}
