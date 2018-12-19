# Build mode readme

## What difference between pool define and pool create as

Objects in libvirt can be either transient or persistent. A transient object only exists for as long as it is running, while a persistent object exists all the time. Essentially with a persistent object the XML configuration is saved by libvirt in /etc/libvirt.

So in the case of storage pools, if you use `virsh pool-define-as` you'll be created a configuration file for a persistent storage pool. You can later start this storage pool using `virsh pool-start`, stop it with `virsh pool-destroy` and start it again later, or even set it to auto-start on host boot.

If you want a transient storage pool, you can use `virsh pool-create-as` which will immediately start a storage pool, without saving its config on disk. This storage pool will totally disappear when you do `virsh pool-destory` (though the actually storage will still exist, libvirt simply won't know about it). Witha transient storage pool, you obviously can't make it auto-start on boot, since libvirt doesn't know about its config.

As a general rule, most people/apps will want to use persistent pools.

Doc link: https://stackoverflow.com/questions/41471006/virsh-difference-between-pool-define-as-and-pool-create-as

## What done of the command `virsh pool-start` ?

The source code of the command as following:

~~~
static bool
cmdPoolStart(vshControl *ctl, const vshCmd *cmd)
{
    virStoragePoolPtr pool;
    bool ret = true;
    const char *name = NULL;
    bool build;
    bool overwrite;
    bool no_overwrite;
    unsigned int flags = 0;

    if (!(pool = virshCommandOptPool(ctl, cmd, "pool", &name)))
         return false;

    build = vshCommandOptBool(cmd, "build");
    overwrite = vshCommandOptBool(cmd, "overwrite");
    no_overwrite = vshCommandOptBool(cmd, "no-overwrite");

    VSH_EXCLUSIVE_OPTIONS_EXPR("overwrite", overwrite,
                               "no-overwrite", no_overwrite);

    if (build)
        flags |= VIR_STORAGE_POOL_CREATE_WITH_BUILD;
    if (overwrite)
        flags |= VIR_STORAGE_POOL_CREATE_WITH_BUILD_OVERWRITE;
    if (no_overwrite)
        flags |= VIR_STORAGE_POOL_CREATE_WITH_BUILD_NO_OVERWRITE;

    if (virStoragePoolCreate(pool, flags) == 0) {
        vshPrintExtra(ctl, _("Pool %s started\n"), name);
    } else {
        vshError(ctl, _("Failed to start pool %s"), name);
        ret = false;
    }

    virStoragePoolFree(pool);
    return ret;
}
~~~

The key point is function `virStoragePoolCreate`

Code link: https://github.com/libvirt/libvirt/blob/4d95d35637e3f59526288e0a8a77f7a200992652/tools/virsh-pool.c#L1726

## virStoragePoolCreate

The document described like this

~~~
virStoragePoolCreate
int	virStoragePoolCreate		(virStoragePoolPtr pool,
					 unsigned int flags)
Starts an inactive storage pool

pool
pointer to storage pool
flags
bitwise-OR of virStoragePoolCreateFlags
Returns
0 on success, or -1 if it could not be started
~~~

Doc link: https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolCreate

`libvirt-go` wrapped it:

~~~
// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolCreate
func (p *StoragePool) Create(flags StoragePoolCreateFlags) error {
	var err C.virError
	result := C.virStoragePoolCreateWrapper(p.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}
~~~

Code link: 
https://github.com/libvirt/libvirt-go/blob/2c9281d85d6de9b8f699c17aceee00e4419b9cc8/storage_pool.go#L117