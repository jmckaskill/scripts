if status is-interactive
    # Commands to run in interactive sessions can go here
end

set -gx EDITOR vim
set -gx PATH $HOME/bin /bin
set -gx SVDIR $HOME/sv
set -gx BROWSER firefox

function fish_greeting
end
