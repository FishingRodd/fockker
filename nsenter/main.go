package nsenter

// Linux 的 setns 系统调用要求在单线程环境中执行，但 Go 语言的运行时（runtime）默认是多线程的（如垃圾回收线程）。直接调用 setns 可能导致未定义行为或失败。
// Docker 通过 CGO 嵌入 C 代码，在 Go 程序初始化前（main 函数执行前）调用 setns，绕过 Go 运行时多线程的限制。
// 这是通过 __attribute__((constructor)) 修饰的 C 函数实现的，该函数一旦被引用，那么这个函数就会被自动执行。

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

__attribute__((constructor)) void enter_namespace(void) {
	// 从环境变量中获取需要进入的PID
	char *TARGET_PID;
	TARGET_PID = getenv("TARGET_PID");
	if (!TARGET_PID) {
		// 如果没有在环境变量中指定PID，则直接退出
		return;
	}
	// 从环境变量中获取需要执行的命令
	char *TARGET_CMD;
	TARGET_CMD = getenv("TARGET_CMD");
	if (!TARGET_CMD) {
		// 如果没有在环境变量中指定PID命令，则直接退出
		return;
	}
	int i;
	char nspath[1024];
	// 定义系统调用属性
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };

	for (i=0; i<5; i++) {
		sprintf(nspath, "/proc/%s/ns/%s", TARGET_PID, namespaces[i]);
		int fd = open(nspath, O_RDONLY);

		if (setns(fd, 0) == -1) {
			//fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			//fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
	int res = system(TARGET_CMD);
	exit(0);
	return;
}
*/
import "C"
