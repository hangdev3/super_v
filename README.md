# super_v

Um clipboard contextual para ajudar usuários a levar hábitos do Windows 11 para o Linux Mint Cinnamon.

O primeiro objetivo é reproduzir o fluxo do `Win+V` com `Super+Shift+V`: popup de histórico próximo ao ponteiro, seleção rápida e cópia do item escolhido para o clipboard. O histórico mistura textos e imagens, tem limite de 10 itens e descarta o mais antigo quando recebe uma nova entrada. O MVP também inclui capturas de área com `Super+Shift+S`, miniaturas de imagens e recuperação dessas imagens pelo clipboard.

O MVP será direcionado a Cinnamon em X11. Wayland e outros atalhos do Windows serão tratados como extensões depois que o núcleo do histórico estiver estável. No X11, a seleção de um item de texto pode enviar `Ctrl+V` automaticamente para a janela que receber o foco externo; imagens são copiadas para o clipboard sem auto-colagem.

A especificação e as decisões arquiteturais estão em [`cloud.md`](cloud.md).

O projeto é distribuído sob a [Apache License 2.0](LICENSE). As notas da
release atual estão em [`CHANGELOG.md`](CHANGELOG.md).

## Desenvolvimento no Linux Mint/Cinnamon

O build gráfico precisa dos headers nativos do Fyne e do hotkey X11:

```bash
sudo apt install \
  libgl1-mesa-dev libx11-dev libxrandr-dev libxinerama-dev \
  libxcursor-dev libxi-dev libxxf86vm-dev libwayland-dev xclip \
  gnome-screenshot libnotify-bin
```

Depois:

```bash
go mod tidy
go run -buildvcs=false ./cmd/super-v
```

Para gerar uma versão instalável no Linux Mint/Cinnamon, use:

```bash
make package-test
sudo apt install ./build/super-v_0.1.0_amd64.deb
```

O pacote instala o executável em `/usr/bin/super-v`, cria o lançador no menu
de aplicativos e registra o autostart em `/etc/xdg/autostart`. Depois de sair e
entrar novamente na sessão, os atalhos globais ficam disponíveis sem iniciar o
binário manualmente. Para gerar outra versão, use `SUPERV_VERSION=0.2.0 make
package-test`.

Se for necessário desativar apenas a inicialização automática para o usuário
atual, copie o arquivo para a configuração pessoal e altere a opção:

```bash
mkdir -p ~/.config/autostart
cp /etc/xdg/autostart/super-v.desktop ~/.config/autostart/super-v.desktop
sed -i 's/^X-GNOME-Autostart-enabled=true/X-GNOME-Autostart-enabled=false/' ~/.config/autostart/super-v.desktop
```

O aplicativo continuará instalado e poderá ser iniciado pelo menu ou com
`/usr/bin/super-v`.

Para testar somente a UI, sem depender do `Super+Shift+V`, execute:

```bash
go run -buildvcs=false ./cmd/super-v --show-on-start
```

Esse modo abre o popup vazio uma vez após a inicialização e também informa no terminal quando o atalho global for recebido.

Enquanto estiver ativo, o aplicativo exibe um ícone de clipboard na bandeja do sistema. O ícone não abre nenhuma janela ao ser clicado; o menu da bandeja oferece somente a ação `Sair`.

O atalho `Super+Shift+S` abre o `gnome-screenshot` no modo de seleção de área. A captura é salva em `~/Imagens/Captures`, copiada para o clipboard como a imagem mais recente, adicionada ao histórico como imagem e sinalizada por uma notificação desktop. Se `libnotify-bin` não estiver instalado, a captura continua sendo salva e o motivo fica registrado no log.

O monitor passivo registra textos novos e imagens copiadas de outros aplicativos, salvando imagens em `~/Imagens/Captures` sem reescrever o clipboard. A conclusão de uma captura e a seleção explícita de um item do histórico são ações do usuário que podem atualizar o clipboard; imagens não acionam auto-colagem. Para textos, o popup fica armado para auto-colagem: clique no campo de destino e, quando a janela externa receber o foco, o aplicativo envia `Ctrl+V` e fecha o popup. Esse recurso depende da extensão XTEST do X11 e é desativado automaticamente quando ela não está disponível; em Wayland não é aplicado.

Seleções X11 que anunciam `image/*` são ignoradas pelo leitor textual, evitando que metadados ou hashes de PNG apareçam como uma segunda linha no histórico. O popup tem linhas de 64 px, miniaturas de 48 px e fundo escuro arredondado com aproximadamente 90% de opacidade.

Para encerrar, pressione `Ctrl+C` no terminal. O aplicativo trata `SIGINT` e `SIGTERM`, fecha o loop do Fyne e libera o atalho global. Se o terminal continuar mostrando `^C` sem voltar ao prompt, localize o processo e envie o encerramento de forma explícita:

```bash
pgrep -af 'super-v|go-build'
kill -TERM <PID_DO_SUPER_V>
```

Para validar em CI sem abrir uma sessão gráfica nem exigir CGO, use o driver de teste do Fyne:

```bash
CGO_ENABLED=0 go test -tags ci ./...
```

As regras para fixtures de imagem e o incidente de consumo extremo de memória estão documentados em [`docs/testing.md`](docs/testing.md).

O build de produção requer uma sessão X11 (`DISPLAY` definido), uma implementação de clipboard (`xclip` ou `xsel`) e os headers acima durante a compilação.

## Publicação de releases

O workflow [`CI`](.github/workflows/ci.yml) executa testes, race detector,
`go vet` e o smoke test do pacote. Ao enviar uma tag no formato `vX.Y.Z`, o
workflow [`Release`](.github/workflows/release.yml) gera o `.deb` e cria a
release do GitHub com o artefato anexado.

Exemplo:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin main --follow-tags
```
