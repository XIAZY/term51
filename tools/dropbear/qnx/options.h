/* Dropbear's options for the QNX build of its client programs
 * (qnx/build.sh copies this to the build folder as localoptions.h,
 * the name Dropbear looks for).
 * Everything not named here is Dropbear's default. */

/* Client programs only. These server options need crypt() or setresgid(),
 * which QNX lacks, and are checked even when no server is built. */
#define DROPBEAR_SVR_PASSWORD_AUTH 0
#define DROPBEAR_SVR_PAM_AUTH 0
#define DROPBEAR_SVR_DROP_PRIVS 0
#define DROPBEAR_SVR_LOCALSTREAMFWD 0

/* scp starts the client by this name, found on the PATH. Dropbear's
 * default is the fixed place /usr/bin/dbclient, which these systems do not
 * have and an app cannot write to. */
#define DROPBEAR_PATH_SSH_PROGRAM "dbclient"

/* For servers as old as QNX 6.5's own sshd (OpenSSH 5.2 of 2009), which
 * knows no other signature, key exchange and integrity check than the
 * SHA-1 ones. Dropbear offers them after the modern ones, so they are used
 * only with a server that has nothing better. */
#define DROPBEAR_RSA_SHA1 1
#define DROPBEAR_DH_GROUP14_SHA1 1
#define DROPBEAR_SHA1_HMAC 1
