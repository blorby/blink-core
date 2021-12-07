#define _GNU_SOURCE

#include <stdio.h>
#include <dlfcn.h>
#include <sys/stat.h>
#include <fcntl.h>


FILE *fopen(const char *path, const char *mode) {
    FILE *(*original_fopen)(const char*, const char*);
    original_fopen = dlsym(RTLD_NEXT, "fopen");
    FILE *res = (*original_fopen)(path, mode);
    printf("### fopen( %s, %si) = %p\n", path, mode, res);
    return res;
}

int open(const char *path, int flags, ...) {
    int (*real_open)() = dlsym(RTLD_NEXT, "open");
    int res = (real_open(path, flags));
    printf("### open( %s, %d) = %d\n", path, flags, res);
    return res;
}

/*
int fclose(FILE *f);
    int (*myreal_fclose)(FILE *) = dlsym(RTLD_NEXT, "fclose");
    FILE *res = (myreal_fclose)(f);
    printf("### fclose( %p) = %d\n", f, res);
    return res;
}
*/

int close(int desc) {
    int (*real_close)() = dlsym(RTLD_NEXT, "close");
    int res = (real_close(desc));
    printf("### close(%d) = %d\n", desc, res);
    return res;
}
