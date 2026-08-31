// MIT. Tiny ABI 1 SCRP tape. Not Godot VM. Not official Luau. No GTE.

#ifndef BLAZIUM_PS2_SCRIPT_VM_H
#define BLAZIUM_PS2_SCRIPT_VM_H

#ifdef __cplusplus
extern "C" {
#endif

int script_vm_init(const unsigned char *scrp, unsigned scrp_sz,
		const unsigned char *node, unsigned node_sz);
void script_vm_process(float delta);
int script_vm_ready(void);
int script_vm_kit_spawn(void);
int script_vm_kit_player(void);
int script_vm_kit_portal(void);
int script_vm_kit_checkpoint(void);
int script_vm_kit_hazard(void);
int script_vm_kit_pickup(void);
int script_vm_kit_load(void);
int script_vm_kit_save(void);

#ifdef __cplusplus
}
#endif

#endif
