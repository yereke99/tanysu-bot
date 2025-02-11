package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"tanysu-bot/config"
	"tanysu-bot/internal/keyboard"
	"tanysu-bot/internal/repository"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Handler содержит все методы-обработчики для бота.
type Handler struct {
	chatState *repository.ChatRepository
	userRepo  *repository.UserRepository
	config    *config.Config
}

func NewHandler(chatState *repository.ChatRepository, userRepo *repository.UserRepository, config *config.Config) *Handler {
	return &Handler{chatState: chatState, userRepo: userRepo, config: config}
}

// ensureUserInDB сохраняет пользователя в БД при первом обращении.
func (h *Handler) ensureUserInDB(update *models.Update) {
	var userID int64
	var username, firstName, lastName string

	if update.Message != nil {
		userID = update.Message.From.ID
		username = update.Message.From.Username
		firstName = update.Message.From.FirstName
		lastName = update.Message.From.LastName
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		username = update.CallbackQuery.From.Username
		firstName = update.CallbackQuery.From.FirstName
		lastName = update.CallbackQuery.From.LastName
	} else {
		return
	}

	exists, err := h.userRepo.UserExists(userID)
	if err != nil {
		fmt.Println("Error checking user existence:", err)
		return
	}
	if !exists {
		newUser := &repository.User{
			UserID:    userID,
			UserName:  username,
			FirstName: firstName,
			LastName:  lastName,
		}
		if err := h.userRepo.InsertUser(newUser); err != nil {
			fmt.Println("Error inserting user:", err)
		} else {
			fmt.Printf("User %d inserted into DB\n", userID)
		}
	}
}

// CheckRegistration проверяет, заполнены ли обязательные поля регистрации.
// Если хотя бы одно из полей не заполнено, регистрация не завершена.
func (h *Handler) CheckRegistration(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	} else {
		return false
	}

	user, err := h.userRepo.GetUser(userID)
	if err != nil {
		fmt.Println("Ошибка получения пользователя:", err)
		return false
	}

	// Если хоть одно из обязательных полей пустое – регистрация не завершена.
	if user.AvaFileID == "" || user.UserNickname == "" || user.UserSex == "" || user.UserAge == 0 || user.UserGeo == "" {
		return false
	}
	return true
}

// RegistrationHandler обрабатывает входящие данные для регистрации пользователя.
// Ожидается, что пользователь отправит фото с подписью в формате:
//
//	@nickname
//	Еркек        (или "Әйел")
//	25
//
// RegistrationHandler обрабатывает входящие данные для регистрации (или обновления) пользователя.
// Ожидается, что пользователь отправит фото с caption в формате:
//
//	@nickname
//	Еркек немесе Әйел
//	25
//
// Если формат неверный, бот отправляет сообщение с примером.
func (h *Handler) RegistrationHandler(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	} else {
		return false
	}

	// Если пользователь уже занят (имеется собеседник), регистрация не требуется.
	partnerID, err := h.chatState.GetUserPartner(ctx, userID)
	if err != nil {
		fmt.Println("Ошибка при получении собеседника:", err)
		return false
	}
	if partnerID != 0 {
		return false
	}

	user, err := h.userRepo.GetUser(userID)
	if err != nil {
		fmt.Println("Ошибка получения пользователя:", err)
		return false
	}

	// Если сообщение не содержит ни фото, ни геолокацию – отправляем инструкцию.
	if update.Message != nil && update.Message.Photo == nil && update.Message.Location == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Өтінеміз, тіркеу үшін төмендегі үлгіге сәйкес фото жіберіп, оған caption ретінде төмендегідей мәліметтерді енгізіңіз:\n\n@nickname\nЕркек немесе Әйел\n25",
		})
		return false
	}

	// Если получено фото с caption (даже если ранее регистрация была выполнена) – обновляем данные.
	if update.Message != nil && update.Message.Photo != nil {
		// Приводим caption к единообразному виду.
		captionText := update.Message.Caption
		captionText = strings.ReplaceAll(captionText, "\r\n", "\n")
		lines := strings.Split(captionText, "\n")
		fmt.Println("Полученные строки из caption:", lines)
		if len(lines) < 3 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Қате формат! Тіркеу үлгісі:\n@nickname\nЕркек немесе Әйел\n25",
			})
			return false
		}
		// Первая строка – лақап аты (удаляем символ @, если есть).
		nickname := strings.TrimSpace(lines[0])
		if strings.HasPrefix(nickname, "@") {
			nickname = strings.TrimPrefix(nickname, "@")
		}
		// Вторая строка – жыныс (ожидаем "Еркек" или "Әйел").
		gender := strings.TrimSpace(lines[1])
		if gender != "Еркек" && gender != "Әйел" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Қате формат! Екінші жолда тек 'Еркек' немесе 'Әйел' болуы тиіс.\nМысал:\n@nickname\nЕркек\n25",
			})
			return false
		}
		// Третья строка – жас (возраст).
		ageStr := strings.TrimSpace(lines[2])
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Қате формат! Жас саны сан түрінде болуы керек, мысалы: 25\nМысал:\n@nickname\nЕркек\n25",
			})
			return false
		}

		// Проверяем наличие директории для аватарок и создаём её, если отсутствует.
		avaDir := "./ava"
		if _, err := os.Stat(avaDir); os.IsNotExist(err) {
			if err := os.MkdirAll(avaDir, os.ModePerm); err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: userID,
					Text:   "Аватар сақталатын директорияны жасау кезінде қате пайда болды.",
				})
				return false
			}
		}
		savedPath := fmt.Sprintf("%s/%d.jpg", avaDir, userID)
		photo := update.Message.Photo[len(update.Message.Photo)-1]
		// Обновляем аватар (сохраняем путь и FileID).
		if err := h.userRepo.UpdateAvatar(userID, savedPath, photo.FileID); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Аватарды жаңарту қатесі: " + err.Error(),
			})
			return false
		}
		// Обновляем никнейм, жыныс и жас.
		if err := h.userRepo.UpdateNickname(userID, nickname); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Никнеймді жаңарту қатесі: " + err.Error(),
			})
			return false
		}
		if err := h.userRepo.UpdateUserSex(userID, gender); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Жынысты жаңарту қатесі: " + err.Error(),
			})
			return false
		}
		if err := h.userRepo.UpdateUserAge(userID, age); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Жас жаңарту қатесі: " + err.Error(),
			})
			return false
		}

		// Составляем summary-подпись.
		summaryCaption := fmt.Sprintf("Лақап атыңыз: @%s\nЖынысыңыз: %s\nЖасыңыз: %d\nЕгер мәліметтеріңізді жаңартқыңыз келсе, қайта жіберіңіз.", nickname, gender, age)
		// Отправляем фото с summary-подписью.
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:         userID,
			Photo:          &models.InputFileString{Data: photo.FileID},
			Caption:        summaryCaption,
			ProtectContent: true,
		})
		// После успешной регистрации (или обновления) отправляем сообщение с кнопкой для геолокации.

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Тіркеу сәтті аяқталды! Орныңызды бөлісу түймесі арқылы партнер таба аласыз!",
		})
		return true
	}

	// Обработка геолокации: если геолокация не заполнена и пришло сообщение с локацией.
	if user.UserGeo == "" && update.Message != nil && update.Message.Location != nil {
		geo := fmt.Sprintf("%.5f,%.5f", update.Message.Location.Latitude, update.Message.Location.Longitude)
		if err := h.userRepo.UpdateUserGeo(userID, geo); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Геолокацияны жаңарту қатесі: " + err.Error(),
			})
			return false
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Геолокация сәтті сақталды! Енді сіз собеседник таба аласыз.",
		})
		return true
	}

	return false
}

func (h *Handler) InlineHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Сначала сохраняем пользователя, если он впервые зашёл.
	h.ensureUserInDB(update)

	// Если callback data равна "send_geo", то обрабатываем запрос на геолокацию.
	if update.CallbackQuery != nil && update.CallbackQuery.Data == "send_geo" {
		// Если нет, можно просто отправить сообщение с инструкцией:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.From.ID,
			Text:   "Өтінеміз, геолокацияңызды жіберіңіз.\n(Мысалы, 'геолокация жіберу' батырмасын немесе 'орныңызды бөлісу' функциясын пайдаланыңыз)",
		})
	}

	// Далее обрабатываем остальные callback'и, например, выбор собеседника.
	if update.CallbackQuery != nil {
		var selectedUserID int64
		_, err := fmt.Sscanf(update.CallbackQuery.Data, "select_%d", &selectedUserID)
		if err != nil {
			fmt.Println("Ошибка при чтении выбранного ID:", err)
			return
		}

		ok, err := h.chatState.CheckPartnerToEmpty(ctx, selectedUserID)
		if err != nil {
			fmt.Println("Ошибка в CheckPartnerToEmpty:", err)
			return
		}
		if ok {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.From.ID,
				Text:   fmt.Sprintf("Собеседник сейчас занят, пожалуйста подождите: %d", selectedUserID),
			})
			return
		}

		if err := h.chatState.SetPartner(ctx, update.CallbackQuery.From.ID, selectedUserID); err != nil {
			fmt.Println("Ошибка в SetPartner:", err)
			return
		}
		if err := h.chatState.SetPartner(ctx, selectedUserID, update.CallbackQuery.From.ID); err != nil {
			fmt.Println("Ошибка в SetPartner (собеседника):", err)
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("Вы подключены к собеседнику с ID: %d", selectedUserID),
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: selectedUserID,
			Text:   fmt.Sprintf("Вы подключены к собеседнику с ID: %d", update.CallbackQuery.From.ID),
		})
	}
}

// CallbackHandlerExit обрабатывает выход пользователя из чата.
func (h *Handler) CallbackHandlerExit(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.ensureUserInDB(update)

	userID := update.CallbackQuery.From.ID
	partnerID, err := h.chatState.GetUserPartner(ctx, userID)
	if err != nil {
		fmt.Println("Ошибка при получении собеседника:", err)
		return
	}

	kb := keyboard.NewKeyboard()
	kb.AddRow(keyboard.NewInlineButton("💬 Chat", "chat"))

	if err := h.chatState.RemoveUser(ctx, userID); err != nil {
		fmt.Println("Ошибка при удалении пользователя:", err)
		return
	}

	if partnerID != 0 {
		if err := h.chatState.RemoveUser(ctx, partnerID); err != nil {
			fmt.Println("Ошибка при удалении собеседника:", err)
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      partnerID,
			Text:        "Ваш собеседник покинул чат.",
			ReplyMarkup: kb.Build(),
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        "Вы вышли из чата.",
		ReplyMarkup: nil,
	})
}

// ChatButtonHandler формирует список пользователей для подключения и отправляет инлайн-кнопки.
func (h *Handler) ChatButtonHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.ensureUserInDB(update)

	userID := update.CallbackQuery.From.ID

	if err := h.chatState.AddUser(ctx, userID); err != nil {
		fmt.Println("Ошибка при добавлении пользователя в чат:", err)
		return
	}

	users, err := h.chatState.GetUsers(ctx)
	if err != nil {
		fmt.Println("Ошибка получения пользователей из чата:", err)
		return
	}

	kb := keyboard.NewKeyboard()
	for _, u := range users {
		if u != userID {
			kb.AddRow(keyboard.NewInlineButton(fmt.Sprintf("User %d", u), fmt.Sprintf("select_%d", u)))
		}
	}

	if len(users) == 1 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Нет доступных пользователей для подключения. Подождите...",
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        "Выберите пользователя для подключения:",
		ReplyMarkup: kb.Build(),
	})
}

// MessageHandler перенаправляет текстовые сообщения между собеседниками.
// Если пользователь свободен и регистрация не завершена, сначала обрабатываются регистрационные данные.
func (h *Handler) MessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.ensureUserInDB(update)

	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	}

	partnerID, err := h.chatState.GetUserPartner(ctx, userID)
	if err != nil {
		fmt.Println("Ошибка получения собеседника:", err)
		return
	}
	if partnerID != 0 {
		h.HandleChat(ctx, b, update, h.chatState)
		return
	}

	if !h.CheckRegistration(ctx, b, update) {
		if h.RegistrationHandler(ctx, b, update) {
			if !h.CheckRegistration(ctx, b, update) {
				return
			}
		} else {
			return
		}
	}

	h.HandleChat(ctx, b, update, h.chatState)
}

// HelloHandler приветствует пользователя и выводит кнопку для входа в чат.
func (h *Handler) HelloHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.ensureUserInDB(update)

	kb := keyboard.NewKeyboard()
	kb.AddRow(keyboard.NewInlineButton("💬 Chat", "chat"))

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Сәлем, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "! Чатқа қосылу үшін '💬 Chat' батырмасын басыңыз.",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb.Build(),
	})
	if err != nil {
		fmt.Println("Ошибка при отправке сообщения:", err)
	}
}

func (h *Handler) DeleteMessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var senderChatID int64
	var senderMsgID int
	var partnerChatID int64
	var partnerMsgID int

	_, err := fmt.Sscanf(update.CallbackQuery.Data, "delete_%d_%d_%d_%d", &senderChatID, &senderMsgID, &partnerChatID, &partnerMsgID)
	if err != nil {
		fmt.Println("Ошибка при извлечении данных из callback:", err)
		return
	}

	okSend, errSender := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    senderChatID,
		MessageID: senderMsgID,
	})
	if errSender != nil {
		fmt.Println("Ошибка при удалении сообщения отправителя:", errSender)
	}

	okPartner, errPartner := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    partnerChatID,
		MessageID: partnerMsgID,
	})
	if errPartner != nil {
		fmt.Println("Ошибка при удалении сообщения собеседника:", errPartner)
	}

	responseChatID := update.CallbackQuery.From.ID
	if !okSend || !okPartner {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: responseChatID,
			Text:   "Хабарлама өшірілмеді!",
		})
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: responseChatID,
		Text:   "Хабарлама сәтті өшірілді!",
	})
}

// HandleChat осуществляет передачу сообщений между собеседниками и пересылает их в канал.
func (h *Handler) HandleChat(ctx context.Context, b *bot.Bot, update *models.Update, chatState *repository.ChatRepository) {
	ForwardChannelID := h.config.ChannelName

	userID := update.Message.From.ID
	partnerID, err := chatState.GetUserPartner(ctx, userID)
	if err != nil {
		fmt.Println("Ошибка при получении собеседника:", err)
		return
	}

	fmt.Printf("[LOG] UserID=%d -> PartnerID=%d | MessageType=", userID, partnerID)

	kbChat := keyboard.NewKeyboard()
	kbChat.AddRow(keyboard.NewInlineButton("💬 Chat", "chat"))

	if partnerID == 0 {
		fmt.Println("Собеседник не найден")
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Сіз әлі сөйлесушімен байланысқа қосылмағансыз. Чатқа қосылу үшін '💬 Chat' батырмасын басыңыз.",
			ReplyMarkup:    kbChat.Build(),
			ProtectContent: true,
		})
		return
	}

	kb := keyboard.NewKeyboard()
	kb.AddRow(keyboard.NewInlineButton("🔕 Шығу", "exit"))

	senderIdentifier := ""
	if update.Message.From.Username != "" {
		senderIdentifier = "@" + update.Message.From.Username
	} else {
		senderIdentifier = fmt.Sprintf("%d", update.Message.From.ID)
	}
	var partnerIdentifier string
	if partnerID != 0 {
		partnerIdentifier = fmt.Sprintf("%d", partnerID)
	} else {
		partnerIdentifier = "сөйлесуші жоқ"
	}

	var caption string
	if update.Message.Caption != "" {
		caption = fmt.Sprintf("%s: %s", senderIdentifier, update.Message.Caption)
	}

	switch {
	case update.Message.Text != "":
		fmt.Printf("TEXT | User=%s | Text=%q\n", senderIdentifier, update.Message.Text)
		partnerMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         partnerID,
			Text:           fmt.Sprintf("%s: %s", senderIdentifier, update.Message.Text),
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("error in send text", err)
			return
		}

		// Отправляем сообщение отправителю с кнопкой удаления.
		// Здесь update.Message.Chat.ID — это чат отправителя.
		senderMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			// Пока клавиатура пустая – далее мы сформируем callback data с обоими ID.
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке сообщения отправителю:", err)
			return
		}

		// Формируем callback data, включающую оба chatID и оба messageID.
		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Хабарламыны жою!", callbackData))

		// Редактируем отправленное сообщение отправителю, добавляя клавиатуру с кнопкой удаления.
		// Если метод редактирования не поддерживается, можно просто отправить новое сообщение.
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ReplyMarkup:    deleteKb.Build(),
			ProtectContent: true,
		})

		textToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, update.Message.Text)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           textToChannel,
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка пересылки сообщения:", err)
		}
	case update.Message.Photo != nil:
		fmt.Printf("PHOTO | User=%s | FileID=%s | Caption=%q\n",
			senderIdentifier,
			update.Message.Photo[len(update.Message.Photo)-1].FileID,
			update.Message.Caption,
		)
		photoID := update.Message.Photo[len(update.Message.Photo)-1].FileID
		partnerMsg, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:         partnerID,
			Photo:          &models.InputFileString{Data: photoID},
			Caption:        withDefaultCaption(senderIdentifier, caption, "фото"),
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке фото собеседнику:", err)
			return
		}
		senderMsg, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:         update.Message.Chat.ID,
			Photo:          &models.InputFileString{Data: photoID},
			Caption:        "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке фото отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Фотоны жою!", callbackData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ReplyMarkup:    deleteKb.Build(),
			ProtectContent: true,
		})

		photoCaption := withDefaultCaption(senderIdentifier, caption, "фото")
		captionToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, photoCaption)
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:         ForwardChannelID,
			Photo:          &models.InputFileString{Data: photoID},
			Caption:        captionToChannel,
			ProtectContent: true,
		})
		// 3. Видео.
	case update.Message.Video != nil:
		fmt.Printf("VIDEO | User=%s | FileID=%s | Caption=%q\n",
			senderIdentifier,
			update.Message.Video.FileID,
			update.Message.Caption,
		)
		videoCaption := withDefaultCaption(senderIdentifier, caption, "видео")
		partnerMsg, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:         partnerID,
			Video:          &models.InputFileString{Data: update.Message.Video.FileID},
			Caption:        videoCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке видео отправителю:", err)
			return
		}
		senderMsg, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:         partnerID,
			Video:          &models.InputFileString{Data: update.Message.Video.FileID},
			Caption:        videoCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке видео отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Видеоны жою!", callbackData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ReplyMarkup:    deleteKb.Build(),
			ProtectContent: true,
		})
		captionToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, videoCaption)
		b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:         ForwardChannelID,
			Video:          &models.InputFileString{Data: update.Message.Video.FileID},
			Caption:        captionToChannel,
			ProtectContent: true,
		})

	// 4. Голосовое сообщение.
	case update.Message.Voice != nil:
		fmt.Printf("VOICE | User=%s | FileID=%s | Caption=%q\n",
			senderIdentifier,
			update.Message.Voice.FileID,
			update.Message.Caption,
		)
		voiceCaption := withDefaultCaption(senderIdentifier, caption, "голосовое сообщение")
		partnerMsg, err := b.SendVoice(ctx, &bot.SendVoiceParams{
			ChatID:         partnerID,
			Voice:          &models.InputFileString{Data: update.Message.Voice.FileID},
			Caption:        voiceCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Дыбыстық хабарламаны отправителю:", err)
			return
		}
		senderMsg, err := b.SendVoice(ctx, &bot.SendVoiceParams{
			ChatID:         partnerID,
			Voice:          &models.InputFileString{Data: update.Message.Voice.FileID},
			Caption:        voiceCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Дыбыстық хабарламаны отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Дыбыстық хабарламаны жою!", callbackData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ReplyMarkup:    deleteKb.Build(),
			ProtectContent: true,
		})
		captionToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, voiceCaption)
		b.SendVoice(ctx, &bot.SendVoiceParams{
			ChatID:         ForwardChannelID,
			Voice:          &models.InputFileString{Data: update.Message.Voice.FileID},
			Caption:        captionToChannel,
			ProtectContent: true,
		})

	// 5. Видео-сообщение (VideoNote).
	case update.Message.VideoNote != nil:
		fmt.Printf("VIDEO_NOTE | User=%s | FileID=%s\n",
			senderIdentifier,
			update.Message.VideoNote.FileID,
		)
		partnerMsg, err := b.SendVideoNote(ctx, &bot.SendVideoNoteParams{
			ChatID:         partnerID,
			VideoNote:      &models.InputFileString{Data: update.Message.VideoNote.FileID},
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Видео хабарламаны сообшение отправителю:", err)
			return
		}
		senderMsg, err := b.SendVideoNote(ctx, &bot.SendVideoNoteParams{
			ChatID:         partnerID,
			VideoNote:      &models.InputFileString{Data: update.Message.VideoNote.FileID},
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Видео хабарламаны отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Видео хабарламаны жою!", callbackData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         update.Message.Chat.ID,
			Text:           "Егер хабарламаны өшіргіңіз келсе, төмендегі батырманы басыңыз.",
			ReplyMarkup:    deleteKb.Build(),
			ProtectContent: true,
		})
		captionToChannel := fmt.Sprintf("Сообщение от %s к %s: Видео сообщение", senderIdentifier, partnerIdentifier)
		b.SendVideoNote(ctx, &bot.SendVideoNoteParams{
			ChatID:         ForwardChannelID,
			VideoNote:      &models.InputFileString{Data: update.Message.VideoNote.FileID},
			ProtectContent: true,
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           captionToChannel,
			ProtectContent: true,
		})

	// 6. Документ.
	case update.Message.Document != nil:
		fmt.Printf("DOCUMENT | User=%s | FileID=%s | Caption=%q\n",
			senderIdentifier,
			update.Message.Document.FileID,
			update.Message.Caption,
		)
		docCaption := withDefaultCaption(senderIdentifier, caption, "документ")
		partnerMsg, err := b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:         partnerID,
			Document:       &models.InputFileString{Data: update.Message.Document.FileID},
			Caption:        docCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Құжатты отправителю:", err)
			return
		}
		senderMsg, err := b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:         partnerID,
			Document:       &models.InputFileString{Data: update.Message.Document.FileID},
			Caption:        docCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Құжатты отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Құжатты жою!", callbackData))
		captionToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, docCaption)
		b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:         ForwardChannelID,
			Document:       &models.InputFileString{Data: update.Message.Document.FileID},
			Caption:        captionToChannel,
			ProtectContent: true,
		})

	// 7. Аудио.
	case update.Message.Audio != nil:
		fmt.Printf("AUDIO | User=%s | FileID=%s | Caption=%q\n",
			senderIdentifier,
			update.Message.Audio.FileID,
			update.Message.Caption,
		)
		audioCaption := withDefaultCaption(senderIdentifier, caption, "аудио")
		partnerMsg, err := b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID:         partnerID,
			Audio:          &models.InputFileString{Data: update.Message.Audio.FileID},
			Caption:        audioCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Аудионы отправителю:", err)
			return
		}
		senderMsg, err := b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID:         partnerID,
			Audio:          &models.InputFileString{Data: update.Message.Audio.FileID},
			Caption:        audioCaption,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Аудионы отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Аудионы жою!", callbackData))

		captionToChannel := fmt.Sprintf("Сообщение от %s к %s:\n%s", senderIdentifier, partnerIdentifier, audioCaption)
		b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID:         ForwardChannelID,
			Audio:          &models.InputFileString{Data: update.Message.Audio.FileID},
			Caption:        captionToChannel,
			ProtectContent: true,
		})

	// 8. Локация.
	case update.Message.Location != nil:
		fmt.Printf("LOCATION | User=%s | Lat=%.5f | Long=%.5f\n",
			senderIdentifier,
			update.Message.Location.Latitude,
			update.Message.Location.Longitude,
		)
		partnerMsg, err := b.SendLocation(ctx, &bot.SendLocationParams{
			ChatID:         partnerID,
			Latitude:       update.Message.Location.Latitude,
			Longitude:      update.Message.Location.Longitude,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Гео-локацияны отправителю:", err)
			return
		}
		senderMsg, err := b.SendLocation(ctx, &bot.SendLocationParams{
			ChatID:         partnerID,
			Latitude:       update.Message.Location.Latitude,
			Longitude:      update.Message.Location.Longitude,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Гео-локацияны отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Гео-локацияны жою!", callbackData))

		locationText := fmt.Sprintf("Сообщение от %s к %s:\nЛокация: %.5f, %.5f",
			senderIdentifier, partnerIdentifier, update.Message.Location.Latitude, update.Message.Location.Longitude)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           locationText,
			ProtectContent: true,
		})

	// 9. Стикер.
	case update.Message.Sticker != nil:
		fmt.Printf("STICKER | User=%s | FileID=%s\n",
			senderIdentifier,
			update.Message.Sticker.FileID,
		)
		partnerMsg, err := b.SendSticker(ctx, &bot.SendStickerParams{
			ChatID:         partnerID,
			Sticker:        &models.InputFileString{Data: update.Message.Sticker.FileID},
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Стикерді отправителю:", err)
			return
		}
		senderMsg, err := b.SendSticker(ctx, &bot.SendStickerParams{
			ChatID:         partnerID,
			Sticker:        &models.InputFileString{Data: update.Message.Sticker.FileID},
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Стикерді отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Стикерді жою!", callbackData))

		b.SendSticker(ctx, &bot.SendStickerParams{
			ChatID:         ForwardChannelID,
			Sticker:        &models.InputFileString{Data: update.Message.Sticker.FileID},
			ProtectContent: true,
		})
		stickerInfo := fmt.Sprintf("Сообщение от %s к %s: Стикер", senderIdentifier, partnerIdentifier)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           stickerInfo,
			ProtectContent: true,
		})

	// 10. Контакт.
	case update.Message.Contact != nil:
		contact := update.Message.Contact
		fmt.Printf("CONTACT | User=%s | Phone=%s | FirstName=%s | LastName=%s\n",
			senderIdentifier,
			contact.PhoneNumber,
			contact.FirstName,
			contact.LastName,
		)
		contactText := fmt.Sprintf("%s отправил(а) контакт:\nТел: %s\nИмя: %s %s",
			senderIdentifier,
			contact.PhoneNumber,
			contact.FirstName,
			contact.LastName,
		)
		partnerMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         partnerID,
			Text:           contactText,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Контактіні отправителю:", err)
			return
		}
		senderMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         partnerID,
			Text:           contactText,
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке Контактіні отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Контактіні жою!", callbackData))

		channelContactText := fmt.Sprintf("Сообщение от %s к %s:\nКонтакт:\nТел: %s\nИмя: %s %s",
			senderIdentifier,
			partnerIdentifier,
			contact.PhoneNumber,
			contact.FirstName,
			contact.LastName,
		)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           channelContactText,
			ProtectContent: true,
		})

	// 11. Опрос.
	case update.Message.Poll != nil:
		poll := update.Message.Poll
		fmt.Printf("POLL | User=%s | Question=%q | Options=%d\n",
			senderIdentifier,
			poll.Question,
			len(poll.Options),
		)
		var pollOptions []models.InputPollOption
		for _, o := range poll.Options {
			pollOptions = append(pollOptions, models.InputPollOption{Text: o.Text})
		}
		partnerMsg, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:         partnerID,
			Question:       poll.Question,
			Options:        pollOptions,
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке опрос отправителю:", err)
			return
		}
		senderMsg, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:         partnerID,
			Question:       poll.Question,
			Options:        pollOptions,
			ProtectContent: true,
		})
		if err != nil {
			fmt.Println("Ошибка при отправке опрос отправителю:", err)
			return
		}

		callbackData := fmt.Sprintf("delete_%d_%d_%d_%d", update.Message.Chat.ID, senderMsg.ID, partnerID, partnerMsg.ID)
		deleteKb := keyboard.NewKeyboard()
		deleteKb.AddRow(keyboard.NewInlineButton("⛔️ Хабарламыны жою опрос!", callbackData))
		pollText := fmt.Sprintf("Сообщение от %s к %s: Опрос\nВопрос: %s",
			senderIdentifier, partnerIdentifier, poll.Question)
		b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:         ForwardChannelID,
			Question:       poll.Question,
			Options:        pollOptions,
			ProtectContent: true,
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         ForwardChannelID,
			Text:           pollText,
			ProtectContent: true,
		})

	// 12. Неизвестный тип сообщения.
	default:
		fmt.Printf("UNKNOWN | User=%s\n", senderIdentifier)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:         userID,
			Text:           "Неизвестный тип сообщения. Попробуйте отправить текст, фото, видео, голосовое сообщение или документ.",
			ReplyMarkup:    kb.Build(),
			ProtectContent: true,
		})
	}
}

// withDefaultCaption формирует подпись для медиа-сообщения, если она отсутствует.
func withDefaultCaption(username, caption, mediaType string) string {
	if caption != "" {
		return caption
	}
	return fmt.Sprintf("@%s отправил(а) %s", username, mediaType)
}
