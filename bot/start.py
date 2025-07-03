# bot/start.py

from bot.imports import ApplicationBuilder, CommandHandler, CallbackQueryHandler
from bot.commands import start, searchlogs, info_command, help_command, add_user, remove_user, add_command, remove_command, broadcast_all, profile_command
from bot.callbacks import callback_query_handler
from bot.config import telbot

def main():
    print("\033[92m\033[0m")
    print("\033[92m======================================== \033[0m")
    print("\033[92m\033[0m")
    print("\033[92m        ┬ ┬┌─┐┬  ┌─┐┌─┐┌┬┐┌─┐\033[0m")
    print("\033[92m        │││├┤ │  │  │ ││││├┤   🐦‍🔥\033[0m")
    print("\033[92m        └┴┘└─┘┴─┘└─┘└─┘┴ ┴└─┘\033[0m")
    print("\033[92m\033[0m")
    print("\033[92m        @GloryLogsBot IS ONLINE...\033[0m")
    print("\033[92m\033[0m")
    print("\033[92m======================================== \033[0m")
    print("\033[92m\033[0m")
    
    app = ApplicationBuilder().token(telbot).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("help", help_command))
    app.add_handler(CommandHandler("search", searchlogs))
    app.add_handler(CommandHandler("info", info_command))
    app.add_handler(CommandHandler("add", add_user))
    app.add_handler(CommandHandler("remove", remove_user))
    app.add_handler(CommandHandler("add",    add_command))
    app.add_handler(CommandHandler("remove", remove_command))
    app.add_handler(CommandHandler("all", broadcast_all))
    app.add_handler(CommandHandler("profile", profile_command))
    app.add_handler(CallbackQueryHandler(callback_query_handler))

    app.run_polling(drop_pending_updates=True)

if __name__ == "__main__":
    main()
