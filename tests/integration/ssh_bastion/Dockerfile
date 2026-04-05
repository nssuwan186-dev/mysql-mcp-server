# Minimal SSH bastion for integration tests (key-based auth only).
FROM alpine:3.19

RUN apk add --no-cache openssh-server openssh-client \
    && mkdir -p /root/.ssh \
    && chmod 0700 /root/.ssh

# Generate deterministic ed25519 host key at build time instead of hardcoding
RUN ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""

# RSA/ECDSA for sshd defaults (ed25519 is fixed above; -A only creates missing types).
RUN ssh-keygen -A

# Create empty authorized_keys to be populated at runtime
RUN touch /root/.ssh/authorized_keys \
    && chmod 0600 /root/.ssh/authorized_keys

# Allow root login with public key only and allow TCP forwarding
RUN echo "PermitRootLogin prohibit-password" >> /etc/ssh/sshd_config \
    && echo "PasswordAuthentication no" >> /etc/ssh/sshd_config \
    && echo "KbdInteractiveAuthentication no" >> /etc/ssh/sshd_config \
    && sed -i 's/^AllowTcpForwarding no/AllowTcpForwarding yes/' /etc/ssh/sshd_config

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
