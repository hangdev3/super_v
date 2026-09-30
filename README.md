# super_v

Um clipboard contextual para ajudar usuários a levar hábitos do Windows 11 para o Linux Mint Cinnamon.

O primeiro objetivo é reproduzir o fluxo do `Win+V` com `Super+V`: popup de histórico textual próximo ao ponteiro, seleção rápida e cópia do item escolhido para o clipboard.

O MVP será direcionado a Cinnamon em X11. Imagens, Wayland e outros atalhos do Windows serão tratados como extensões depois que o núcleo do histórico estiver estável. No X11, a seleção de um item pode enviar `Ctrl+V` automaticamente para a janela que receber o foco externo.

A especificação e as decisões arquiteturais estão em [`cloud.md`](cloud.md).

## Desenvolvimento no Linux Mint/Cinnamon

O build gráfico precisa dos headers nativos do Fyne e do hotkey X11:

```bash
sudo apt install \
  libgl1-mesa-dev libx11-dev libxrandr-dev libxinerama-dev \
  libxcursor-dev libxi-dev libxxf86vm-dev libwayland-dev xclip
```

Depois:

```bash
go mod tidy
go run -buildvcs=false ./cmd/super-v
```

Para testar somente a UI, sem depender do `Super+V`, execute:

```bash
go run -buildvcs=false ./cmd/super-v --show-on-start
```

Esse modo abre o popup vazio uma vez após a inicialização e também informa no terminal quando o atalho global for recebido.

Ao selecionar um item, ele é copiado para o clipboard e o popup fica armado para auto-colagem. Clique no campo de destino: quando a janela externa receber o foco, o aplicativo envia `Ctrl+V` e fecha o popup. Esse recurso depende da extensão XTEST do X11 e é desativado automaticamente quando ela não está disponível; em Wayland não é aplicado.

Para encerrar, pressione `Ctrl+C` no terminal. O aplicativo trata `SIGINT` e `SIGTERM`, fecha o loop do Fyne e libera o atalho global. Se o terminal continuar mostrando `^C` sem voltar ao prompt, localize o processo e envie o encerramento de forma explícita:

```bash
pgrep -af 'super-v|go-build'
kill -TERM <PID_DO_SUPER_V>
```

Para validar em CI sem abrir uma sessão gráfica nem exigir CGO, use o driver de teste do Fyne:

```bash
CGO_ENABLED=0 go test -tags ci ./...
```

O build de produção requer uma sessão X11 (`DISPLAY` definido), uma implementação de clipboard (`xclip` ou `xsel`) e os headers acima durante a compilação.
