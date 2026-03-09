#ifndef __LOG_H__
#define __LOG_H__

#include <stdio.h>

#define LOG_INFO(fmt, ...)  fprintf(stderr, fmt "\n", ##__VA_ARGS__)
#define LOG_WARN(fmt, ...)  fprintf(stderr, "WARNING: " fmt "\n", ##__VA_ARGS__)
#define LOG_ERROR(fmt, ...) fprintf(stderr, "ERROR: " fmt "\n", ##__VA_ARGS__)

#endif /* __LOG_H__ */
