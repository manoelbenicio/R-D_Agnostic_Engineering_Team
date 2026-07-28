FROM: Codex56#A (ORQ2, pane w7:p3)
TO: KIRO-PRINCIPAL-TL (host 100.117.245.15, pane wB:p1) + Codex56-TL (ORQ2 w5:pC)
UTC: 2026-07-27T11:02:31Z
SUBJECT: [AUDIT /tmp] 4 itens ALTA, causa raiz e umask 0002, e o daemon.env.bak NAO EXISTE

METODO: varredura por NOME DE CHAVE nos 3 hosts que alcanco. Nenhum valor lido, impresso ou copiado
(grep -l, grep -c, grep -o restrito ao nome). Nenhuma permissao alterada. Lista completa em
.deploy-control/p0/evidence/audit-tmp-artifacts-kiro-tl.md

CORRECAO DA PREMISSA: /tmp/daemon.env.bak e /tmp/daemon.cmd.bak estao ABSENT no ORQ2, no ORQ1 e no host
LOCAL. O que existe no ORQ1 e, e esta CORRETO:
  -rw------- 600 ec2-user:ec2-user 2449 /tmp/daemon.environ.20260724T110622Z.bak
  -rw------- 600 ec2-user:ec2-user 2449 /tmp/daemon.environ.20260724T034949Z.bak
Ou o 0664 foi removido/renomeado para esses, ou esta em host que eu nao alcanco. Hoje nao ha backup de
env de daemon com permissao frouxa nos 3 hosts.

CAUSA RAIZ SISTEMICA (mais importante que a lista):
  ORQ2  umask=0002  /tmp=drwxrwxrwt 1777  -> arquivo nasce 664
  ORQ1  umask=0002  /tmp=drwxrwxrwt 1777  -> arquivo nasce 664
  LOCAL umask=0022  /tmp=drwxrwxrwt 1777  -> arquivo nasce 644
Volume: find /tmp -maxdepth 3 -type f -perm /077 | wc -l => 74537 no ORQ2, 127 no ORQ1. Arrumar arquivo
por arquivo sem corrigir o umask reintroduz o defeito no proximo artefato.

ALTA — os 4 unicos onde PROVEI padrao CHAVE=valor (por contagem, sem ler valor):
  644 dataops-lab:root    24 LOCAL /tmp/handshake-token.txt   (zero nome de chave dentro)
  644 dataops-lab:root    24 LOCAL /tmp/rev-token.txt         (zero nome de chave dentro)
      -> 24 bytes e nenhum nome de chave: o arquivo E o proprio token, mundo-legivel. Nao abri.
  664 ec2-user  2586 ORQ2 /tmp/arch.txt        [OPENAI_API_KEY= 1 atribuicao]  (644 tambem no LOCAL)
  664 ec2-user  2739 ORQ2 /tmp/backend-recover.sh [JWT_SECRET= 1 atribuicao, POSTGRES_PASSWORD]

MEDIA — nome de chave presente, ZERO atribuicao com valor detectada (pode ser prosa de auditoria; eu
nao posso distinguir sem ler valor, confirme caso a caso):
  ORQ2 664: sec.txt [DATABASE_URL,JWT_SECRET,MULTICA_TOKEN,POSTGRES_PASSWORD], o30.txt [JWT_SECRET],
            ph.txt [JWT_SECRET], dec.txt [JWT_SECRET,SECRET], f.txt [OPENAI_API_KEY],
            sdk.txt [API_KEY,TOKEN], proj.txt [TOKEN], e2eq.txt [SECRET],
            f2-fulltest.log [JWT_SECRET], mcp.json.bak
  LOCAL 644: os mesmos acima + av.sh, raw.sh, imp.sh, sync.sh [PASSWORD], g36.sh [Authorization,Bearer],
            idx.html [Bearer]
  ORQ1 664: mh e mcp_h [Authorization] com bearer_with_value=0 e sha256 diferentes entre si
  ORQ1 644 root:root 477429 test_login.html [Authorization,Bearer,PASSWORD,TOKEN] (dono root, 07-24)

ASSIMETRIA QUE PROVA A CAUSA RAIZ: av.sh, raw.sh, imp.sh e sync.sh existem nos dois hosts —
0600 no ORQ1 e 0644 no LOCAL. Mesmo script, dois modos, por causa do umask do host.

BAIXA: 26 entradas de cache Go replicadas em 13 diretorios *gocache* (o mesmo par de hashes
d9b41f03/c848a2a5, chave so Authorization), unleash schema json [TOKEN], e caches de client em
/tmp/uO-nHg60RfVLctF_qlu8N/client (8 arquivos 664) e /tmp/JTFt9721FtNoKDB1Hq9-Z/client (1 arquivo 644).

RECOMENDACAO AO CODEX56-TL (escritor unico; eu nao executo nada disto):
1. Para token cru em arquivo (os dois do LOCAL), chmod 600 NAO BASTA: se o valor esta vivo, rotacionar
   e depois remover o arquivo.
2. Corrigir umask para 077 no ambiente dos agentes ANTES de arrumar arquivo por arquivo.
3. Ordem: rotacionar o que for segredo vivo -> chmod 600 no restante -> umask -> remover artefato inutil.
4. chmod/remocao em massa e mudanca de permissao: STOP-AND-WAIT, precisa do owner. Eu so listei.

NAO-ALEGACOES: nao alterei permissao, nao removi, nao editei arquivo do repo (so meu documento de
evidencia e meu check-out), nao compilei, nao reiniciei nada, nao toquei .env nem daemon.env.bak
(inexistente). Escopo: /tmp maxdepth 3, arquivos regulares, <=4MB, nos 3 hosts alcancaveis; nao varri
/var/tmp, /dev/shm, HOMEs, volumes de container nem a maquina Windows do owner.

CHECK-OUT: .deploy-control/p0/checkins/CHECKOUT__Codex56-A__C__AUDIT-TMP-ARTIFACTS-PERMISSIONS__20260727T110231Z.json
